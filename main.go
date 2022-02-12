package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	_ "github.com/go-sql-driver/mysql"
)

var features = map[string]bool{}

var versionString = "v1.2.0"

//go:embed templates/**
var tmplFS embed.FS

//go:embed static/**
var staticFS embed.FS

var db *sql.DB
var dbErr error

var myConfig conf
var minerList = make(map[string]mineRpc)
var progStartTime = time.Now()
var mutex = &sync.Mutex{}
var totalHashG = ""
var currentPricePerDMO = 0.0

func main() {
	gin.SetMode(gin.ReleaseMode)

	if features["FREE"] { // Don't want the console filled with gin garbage on the free version
		gin.DefaultWriter = ioutil.Discard
	}

	router := gin.Default()

	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{MaxAge: 60 * 60 * 24}) // expire in a day
	router.Use(sessions.Sessions("mysession", store))
	router.Use(sessionMgr())

	myConfig.getConf()
	if myConfig.QuietMode {
		fmt.Printf("Starting monitor in quiet mode (no console output). Access stats at http://localhost:%s/stats\n", myConfig.ServerPort)
	}

	if features["MANAGEMENT"] {
		db, dbErr = sql.Open("mysql", myConfig.DBUser+":"+myConfig.DBPass+"@tcp("+myConfig.DBIP+":"+myConfig.DBPort+")/"+myConfig.DBName)

		// Truly a fatal error.
		if dbErr != nil {
			panic(dbErr.Error())
		}
		defer db.Close()
		fmt.Printf("Connected to DB: %s\n", myConfig.DBName)
		getAllUserInfo()
	}

	if myConfig.MinerLateTime < 20 {
		myConfig.MinerLateTime = 20
	}
	if myConfig.DailyStatDays < 2 {
		myConfig.DailyStatDays = 3
	}
	// Don't let people get too nuts here
	if myConfig.DailyStatDays > 21 {
		myConfig.DailyStatDays = 21
	}
	if myConfig.AutoRefreshSeconds < 10 && myConfig.AutoRefreshSeconds > 0 {
		myConfig.AutoRefreshSeconds = 10
	}

	if len(myConfig.AddrsToMonitor) > 0 {
		txStats()
	}

	if !features["COMINGSOON"] {
		go func() {
			for {
				getCoinGeckoDMOPrice()
				if len(myConfig.AddrsToMonitor) > 0 {
					txStats()
				}

				updateMinerStatus()
				if !myConfig.QuietMode {
					consoleOutput()
				}
				time.Sleep(10 * time.Second)
			}
		}()
	}

	// Anyone can access this route, and it doesn't matter what their session has in it...
	router.StaticFS("/static", myStaticFS())
	templ := template.Must(template.New("").ParseFS(tmplFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	if features["COMINGSOON"] || features["MANAGEMENT"] {
		router.GET("/", landingPage)
	}

	if features["FREE"] {
		router.GET("/stats", statsPage)              // Route that only a logged in user session should be able to access
		router.POST("/minerstats", getMinerStatsRPC) // Route that will need a bearer token from the miner
		router.POST("/removeminer", removeLateMiner) // Route that will need a bearer token from the miner
	}

	if features["MANAGEMENT"] {
		router.GET("/stats", checkLoggedIn(), statsPage) // Route that only a logged in user session should be able to access
		router.GET("/account", checkLoggedIn(), accountPage)
		router.POST("/minerstats", checkBearer(), getMinerStatsRPC) // Route that will need a bearer token from the miner
		router.POST("/removeminer", checkBearer(), removeLateMiner) // Route that will need a bearer token from the miner
	}

	if features["MANAGEMENT"] {
		router.GET("/login", loginPage)
		router.POST("/login", doLogin)
		router.POST("/register", doRegister)
		router.GET("/logout", doLogout)

	}

	router.Run(":" + myConfig.ServerPort)
}

func checkBearer() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.Request.Header.Get("Authorization")
		if auth == "" {
			log.Printf("Attempt to access Bearer auth path with no token\n")
			c.String(http.StatusForbidden, "No Authorization header provided")
			c.Abort()
			return
		} else {
			token := strings.TrimPrefix(auth, "Bearer ")
			_, ok := cloudKeyList[token]
			if !ok {
				log.Printf("Bearer token provided is not authorized: %s\n", token)
				c.String(http.StatusForbidden, "Invalid Token")
				c.Abort()
				return
			}
			c.Set("cloudKey", token)
		}
		c.Next()
	}
}

func sessionMgr() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get a new cookie or set one if it does not exist:
		session := sessions.Default(c)
		id := session.Get("ID")
		guest := session.Get("guest")

		if id == nil {
			session.Set("ID", 0)
		}
		if guest == nil {
			session.Set("guest", true)
		}
		session.Save()

		c.Next()
	}
}

func checkLoggedIn() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get a new cookie or set one if it does not exist:
		session := sessions.Default(c)
		id := session.Get("ID")
		guest := session.Get("guest")
		if id != nil && id.(int) != 0 {
			if guest != nil && !guest.(bool) {
				c.Next()
			}
		}
		c.Redirect(http.StatusTemporaryRedirect, "/")

	}
}

func myStaticFS() http.FileSystem {
	sub, err := fs.Sub(staticFS, "static")

	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

func updateMinerStatus() {
	if len(minerList) > 0 {
		names := make([]string, 0)
		for name, _ := range minerList {
			names = append(names, name)
		}

		sort.Strings(names)

		totalHash := 0
		for _, name := range names {
			stats := minerList[name]
			howLong := time.Now().Sub(stats.LastReport).Round(time.Second)
			stats.HowLate = howLong.String()
			mutex.Lock()
			minerList[name] = stats
			mutex.Unlock()
			if howLong.Seconds() > myConfig.MinerLateTime && stats.Late == false {
				stats.Late = true
				mutex.Lock()
				minerList[name] = stats
				mutex.Unlock()
				if len(myConfig.TelegramUserId) > 0 {
					sendOfflineNotificationToTelegram(name)
				}
			} else if howLong.Seconds() <= myConfig.MinerLateTime {
				stats.Late = false
				mutex.Lock()
				minerList[name] = stats
				mutex.Unlock()
				totalHash += stats.Hashrate
			}
		}
		totalHashG = formatHashNum(totalHash)
	}

	nDays := len(addrStats.DailyStats)
	overallInfoTX.CurrentCoinsPerDay = addrStats.DailyStats[nDays-2].Coins
	overallInfoTX.Projection = fmt.Sprintf("%.1f", addrStats.ProjectedCoinsToday)
	overallInfoTX.NetHash = formatHashNum(int(addrStats.NetHash))

	// Daily Average is the Average of the average of all days except today...
	tmpCoins := 0.0
	tmpPerc := 0.0
	dayStatsTX = nil
	for i := 0; i < nDays; i++ {

		var thisDay DayStatTX
		thisDay.Day = addrStats.DailyStats[i].Day
		thisDay.CoinCount = addrStats.DailyStats[i].Coins
		thisDay.WinPercent = addrStats.DailyStats[i].WinPercent
		thisDay.CoinsPerHour = thisDay.CoinCount / 24.0

		if i < (nDays - 1) {
			tmpCoins += addrStats.DailyStats[i].Coins
			tmpPerc += addrStats.DailyStats[i].WinPercent
		} else {
			thisDay.CoinsPerHour = addrStats.ProjectedCoinsToday / 24.0
		}

		dayStatsTX = append(dayStatsTX, thisDay)

	}
	overallInfoTX.DailyAverage = tmpCoins / (float64(nDays) - 1.0)
	overallInfoTX.HourlyAverage = overallInfoTX.DailyAverage / 24.0
	overallInfoTX.WinPercent = tmpPerc / (float64(nDays) - 1.0)

	nHours := len(addrStats.HourlyStats)

	hourStatsTX = nil
	for j := 0; j < nHours; j++ {
		var thisHour HourStatTX
		thisHour.Hour = addrStats.HourlyStats[j].Hour
		thisHour.CoinCount = float64(addrStats.HourlyStats[j].Coins)
		thisHour.CoinsPerMinute = float64(addrStats.HourlyStats[j].Coins) * (1.0 / 60.0)
		hourStatsTX = append(hourStatsTX, thisHour)
	}

}

// TODO: Add console output back in
func consoleOutput() {

	var H1Col = colorBrightCyan
	var H2Col = colorBrightGreen
	var H3Col = colorBrightBlue
	var WarnCol = colorRed

	fmt.Print("\033[H\033[2J") // Clear screen
	setColor(colorBrightWhite)
	fmt.Printf("\t\t\t\t\t\tDMO-Monitor %s\n\n", versionString)

	setColor(H1Col)
	fmt.Printf("\t\t\t\t\tMiner Monitoring Stats\n\n")

	setColor(H2Col)
	fmt.Printf("\t%s%s%s%s%s%s\n",
		StringSpaced("Miner Name", " ", 24),
		StringSpaced("Last Reported", " ", 35),
		StringSpaced("Hashrate", " ", 12),
		StringSpaced("Submitted", " ", 12),
		StringSpaced("Accepted", " ", 12),
		StringSpaced("Rejected", " ", 12))

	warnings := ""
	if len(minerList) > 0 {
		setColor(H3Col)
		names := make([]string, 0)
		for name, _ := range minerList {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			stats := minerList[name]
			setColor(H3Col)

			if stats.Late {
				warnings += "\n\tWARN: " + name + " has not reported in " + stats.HowLate + "\n"
				setColor(WarnCol)
			}

			fmt.Printf("\t%s%s%s%s%s%s\n",
				StringSpaced(name, " ", 24),
				StringSpaced(stats.LastReport.Format("2006-01-02 15:04:05"), " ", 35),
				StringSpaced(stats.HashrateStr, " ", 12),
				StringSpaced(strconv.Itoa(stats.Submit), " ", 12),
				StringSpaced(strconv.Itoa(stats.Accept), " ", 12),
				strconv.Itoa(stats.Reject),
			)
			setColor(H3Col)
		}

		fmt.Printf("\n\t%s%s\n",
			StringSpaced(fmt.Sprintf("Total Miners: %d", len(minerList)), " ", 32),
			StringSpaced(fmt.Sprintf("Total Hashrate: %s", totalHashG), " ", 32))
	} else {
		setColor(WarnCol)
		fmt.Printf("\t\t\t\tNo active miners\n")
	}

	if len(warnings) > 0 {
		setColor(WarnCol)
		fmt.Printf(warnings)
		setColor(H3Col)
	}

	if myConfig.AddrsToMonitor != "" {

		setColor(H1Col)
		fmt.Printf("\n\n\t\t\t\t\tAddress Mining Stats for Address(es)\n")
		setColor(H2Col)
		fmt.Printf("\n\t\t\t\tDaily Statistics (Last %d Days)\n", myConfig.DailyStatDays)
		fmt.Printf("\t%s%s%s%s\n",
			StringSpaced("Day", " ", 24),
			StringSpaced("Coins", " ", 35),
			StringSpaced("Coins/Hr", " ", 12),
			StringSpaced("Win Percent", " ", 12))
		setColor(H3Col)

		for _, day := range dayStatsTX {
			fmt.Printf("\t%s%s%s%s\n",
				StringSpaced(day.Day, " ", 24),
				StringSpaced(fmt.Sprintf("%.2f", day.CoinCount), " ", 35),
				StringSpaced(fmt.Sprintf("%.2f", day.CoinsPerHour), " ", 12),
				StringSpaced(fmt.Sprintf("%.2f", day.WinPercent), " ", 12))
		}

		setColor(H2Col)
		fmt.Printf("\n\t\t\t\tTodays Hourly Statistics\n")
		fmt.Printf("\t%s%s%s\n",
			StringSpaced("Hour", " ", 24),
			StringSpaced("Coins", " ", 35),
			StringSpaced("Coins/Min", " ", 12))

		setColor(H3Col)

		for _, hour := range hourStatsTX {
			fmt.Printf("\t%s%s%s\n",
				StringSpaced(fmt.Sprintf("%d", hour.Hour), " ", 24),
				StringSpaced(fmt.Sprintf("%.2f", hour.CoinCount), " ", 35),
				StringSpaced(fmt.Sprintf("%.2f", hour.CoinsPerMinute), " ", 12))
		}

		fmt.Printf("\n\t%s\n",
			StringSpaced(fmt.Sprintf("Projected Coins Today: %s", overallInfoTX.Projection), " ", 40))

	} else {
		setColor(WarnCol)
		fmt.Printf("\n\n\t\t\t\t\tNo Receiving Address Statistics Available\n\n")
	}

}
