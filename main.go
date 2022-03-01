package main

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	_ "github.com/go-sql-driver/mysql"
)

var versionString = "v1.2.0"

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

	router := gin.Default()

	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{MaxAge: 60 * 60 * 24 * 7}) // expire in 7 days
	router.Use(sessions.Sessions("mysession", store))
	router.Use(sessionMgr())

	myConfig.getConf()
	fmt.Printf("Access stats at http://localhost:%s/stats\n", myConfig.ServerPort)

	db, dbErr = sql.Open("mysql", myConfig.DBUser+":"+myConfig.DBPass+"@tcp("+myConfig.DBIP+":"+myConfig.DBPort+")/"+myConfig.DBName)

	// Truly a fatal error.
	if dbErr != nil {
		log.Printf("Unable to connect to DB: %s", dbErr.Error())
		os.Exit(1)
	}
	defer db.Close()
	fmt.Printf("Connected to DB: %s\n", myConfig.DBName)
	getAllUserInfo()

	if myConfig.MinerLateTime < 20 {
		myConfig.MinerLateTime = 20
	}
	if myConfig.DailyStatDays < 2 {
		myConfig.DailyStatDays = 3
	}
	if myConfig.DailyStatDays > 21 {
		myConfig.DailyStatDays = 21
	}
	if myConfig.AutoRefreshSeconds < 10 && myConfig.AutoRefreshSeconds > 0 {
		myConfig.AutoRefreshSeconds = 10
	}

	getCoinGeckoDMOPrice()
	txStats()
	updateMinerStatus()

	go func() {
		for {
			getCoinGeckoDMOPrice()
			time.Sleep(240 * time.Second)
		}
	}()

	go func() {
		for {
			txStats()
			time.Sleep(60 * time.Second)
		}
	}()

	go func() {
		for {
			updateMinerStatus()
			time.Sleep(10 * time.Second)
		}
	}()

	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*.html")

	router.GET("/", addPageVars(), landingPage)
	router.GET("/dmowrapversioncheck", getDMOWrapVersion)
	router.GET("/wrapminer", addPageVars(), wrapMiner)
	router.GET("/faq", addPageVars(), faqPage)
	router.GET("/stats", checkLoggedIn(), addPageVars(), statsPage)
	router.GET("/account", checkLoggedIn(), addPageVars(), accountPage)
	router.GET("/admin", checkLoggedIn(), addPageVars(), adminPage)
	router.POST("/changepass", checkLoggedIn(), addPageVars(), doChangePass)
	router.POST("/minerstats", checkBearer(), getMinerStatsRPC)
	router.POST("/removeminer", checkLoggedIn(), addPageVars(), removeLateMiner)
	router.GET("/login", addPageVars(), loginPage)
	router.POST("/login", addPageVars(), doLogin)
	router.POST("/register", addPageVars(), doRegister)
	router.POST("/doupdatetelegramid", checkLoggedIn(), addPageVars(), doUpdateTelegramID)
	router.POST("/doupdateaddrs", checkLoggedIn(), addPageVars(), doUpdateAddrs)
	router.GET("/logout", checkLoggedIn(), addPageVars(), doLogout)

	log.Printf("Starting server!\n")
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

func addPageVars() gin.HandlerFunc {
	return func(c *gin.Context) {
		var pVars pageVars
		session := sessions.Default(c)
		userID := session.Get("ID").(int)
		pVars.UserID = userID
		pVars.UserName = userIDList[userID].UserName
		pVars.CloudKey = html.EscapeString(userIDList[userID].CloudKey)
		pVars.TelegramUserID = userIDList[userID].TelegramUserId
		pVars.Guest = session.Get("guest").(bool)
		pVars.Admin = userIDList[userID].Admin
		pVars.Paid = userIDList[userID].Paid

		c.Set("pVars", pVars)
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
		mutex.Lock()
		lastActive[id.(int)] = time.Now().Format(time.UnixDate)
		mutex.Unlock()

		c.Redirect(http.StatusTemporaryRedirect, "/")

	}
}

func updateMinerStatus() {
	for userID, stats := range addrStats {
		var thisInfo OverallInfoTX

		nDays := len(stats.DailyStats)
		thisInfo.CurrentCoinsPerDay = stats.DailyStats[nDays-2].Coins
		thisInfo.Projection = fmt.Sprintf("%.1f", stats.ProjectedCoinsToday)
		thisInfo.NetHash = formatHashNum(int(stats.NetHash))

		// Daily Average is the Average of the average of all days except today...
		tmpCoins := 0.0
		tmpPerc := 0.0
		for i := 0; i < nDays; i++ {

			var thisDay DayStatTX
			thisDay.Day = stats.DailyStats[i].Day
			thisDay.CoinCount = stats.DailyStats[i].Coins
			thisDay.WinPercent = stats.DailyStats[i].WinPercent
			thisDay.CoinsPerHour = thisDay.CoinCount / 24.0

			if i < (nDays - 1) {
				tmpCoins += stats.DailyStats[i].Coins
				tmpPerc += stats.DailyStats[i].WinPercent
			} else {
				thisDay.CoinsPerHour = stats.ProjectedCoinsToday / 24.0
			}

			thisInfo.DayStats = append(thisInfo.DayStats, thisDay)

		}
		thisInfo.DailyAverage = tmpCoins / (float64(nDays) - 1.0)
		thisInfo.HourlyAverage = thisInfo.DailyAverage / 24.0
		thisInfo.WinPercent = tmpPerc / (float64(nDays) - 1.0)

		nHours := len(stats.HourlyStats)

		thisMinute := float64(time.Now().Minute() + 1)
		for j := 0; j < nHours; j++ {
			var thisHour HourStatTX
			thisHour.Hour = stats.HourlyStats[j].Hour
			thisHour.CoinCount = float64(stats.HourlyStats[j].Coins)
			if j < (nHours - 1) {
				thisHour.CoinsPerMinute = float64(stats.HourlyStats[j].Coins) * (1.0 / 60.0)
			} else {
				thisHour.CoinsPerMinute = float64(stats.HourlyStats[j].Coins) * (1.0 / thisMinute)
			}
			thisInfo.HourStats = append(thisInfo.HourStats, thisHour)
		}

		mutex.Lock()
		overallInfoTX[userID] = thisInfo
		mutex.Unlock()
	}

	mutex.Lock() // For now... lock for the whole time we are reading and writing back to minerList...
	if len(minerList) > 0 {
		for cloudKey, myMinerList := range minerList {

			totalHash := 0
			totalActiveMiners := 0
			for minerID, stats := range myMinerList {
				howLong := time.Since(stats.LastReport).Round(time.Second)
				stats.HowLate = howLong.String()
				myMinerList[minerID] = stats
				if howLong.Seconds() > myConfig.MinerLateTime && !stats.Late {
					stats.Late = true
					myMinerList[minerID] = stats
					if len(cloudKeyList[cloudKey].TelegramUserId) > 0 {
						sendOfflineNotificationToTelegram(stats.Name, cloudKeyList[cloudKey].TelegramUserId)
					}
				} else if howLong.Seconds() <= myConfig.MinerLateTime {
					stats.Late = false
					totalActiveMiners += 1
					myMinerList[minerID] = stats
					totalHash += stats.Hashrate
				}
			}

			userID := cloudKeyList[cloudKey].ID
			var myOverallInfoTX OverallInfoTX = overallInfoTX[userID]

			myOverallInfoTX.TotalActiveMiners = totalActiveMiners
			overallInfoTX[userID] = myOverallInfoTX

			totalHashG[cloudKey] = formatHashNum(totalHash) // TODO: should be by cloudkey...
		}
	}
	mutex.Unlock()

}
