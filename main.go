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
var minerList = make(map[string]map[string]mineRpc)
var progStartTime = time.Now()
var mutex = &sync.Mutex{}
var totalHashG = make(map[string]string)
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
				time.Sleep(10 * time.Second)
			}
		}()
	}

	router.StaticFS("/static", myStaticFS())
	templ := template.Must(template.New("").ParseFS(tmplFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	if features["COMINGSOON"] || features["MANAGEMENT"] {
		router.GET("/", landingPage)
	}

	if features["FREE"] {
		router.GET("/stats", statsPage)
		router.POST("/minerstats", getMinerStatsRPC)
		router.POST("/removeminer", removeLateMiner)
	}

	if features["MANAGEMENT"] {
		router.GET("/stats", checkLoggedIn(), statsPage)
		router.GET("/account", checkLoggedIn(), accountPage)
		router.POST("/changepass", checkLoggedIn(), doChangePass)
		router.POST("/minerstats", checkBearer(), getMinerStatsRPC)
		router.POST("/removeminer", checkLoggedIn(), removeLateMiner)
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

	mutex.Lock() // For now... lock for the whole time we are reading and writing back to minerList...
	if len(minerList) > 0 {
		for cloudKey, myMinerList := range minerList {

			totalHash := 0
			for name, stats := range myMinerList {
				howLong := time.Since(stats.LastReport).Round(time.Second)
				stats.HowLate = howLong.String()
				myMinerList[name] = stats
				if howLong.Seconds() > myConfig.MinerLateTime && !stats.Late {
					stats.Late = true
					myMinerList[name] = stats
					if len(myConfig.TelegramUserId) > 0 {
						sendOfflineNotificationToTelegram(name)
					}
				} else if howLong.Seconds() <= myConfig.MinerLateTime {
					stats.Late = false
					myMinerList[name] = stats
					totalHash += stats.Hashrate
				}
			}
			totalHashG[cloudKey] = formatHashNum(totalHash) // TODO: should be by cloudkey...
		}
	}
	mutex.Unlock()

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
