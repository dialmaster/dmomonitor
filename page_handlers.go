package main

import (
	"html"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type pageVars struct {
	Uptime             string
	MinerList          map[string]mineRpc
	Totalhash          string
	Totalminers        int
	WalletOverallStats OverallInfoTX
	WalletDailyStats   []DayStatTX
	WalletHourlyStats  []HourStatTX
	AutoRefresh        int
	DailyStatDays      int
	VersionString      string
	CurrentPrice       float64
	DollarsPerDay      float64
	DollarsPerWeek     float64
	DollarsPerMonth    float64
	NetHash            string
	PageTitle          string
	Guest              bool
	CloudKey           string
	UserName           string
	Errors             []string
	Addresses          string
	TelegramUserID     string
}

func accountPage(c *gin.Context) {
	var pVars pageVars
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	pVars.PageTitle = "DMO Monitor - My Account"
	pVars.UserName = userIDList[userID].UserName
	pVars.CloudKey = html.EscapeString(userIDList[userID].CloudKey)
	pVars.TelegramUserID = userIDList[userID].TelegramUserId
	errInterface, found := c.Get("errors")
	if found {
		pVars.Errors = errInterface.([]string)
	}

	pVars.Addresses = ""
	for _, address := range userIDList[userID].ReceivingAddresses {
		pVars.Addresses += address.ReceivingAddress + ","
	}
	if len(pVars.Addresses) > 1 {
		pVars.Addresses = pVars.Addresses[:len(pVars.Addresses)-1]
	}

	c.HTML(http.StatusOK, "account.html", pVars)
}

func loginPage(c *gin.Context) {
	session := sessions.Default(c)
	var pVars pageVars
	pVars.PageTitle = "DMO Monitor - Login"
	pVars.Guest = session.Get("guest").(bool)

	errInterface, found := c.Get("errors")
	if found {
		pVars.Errors = errInterface.([]string)
	}

	c.HTML(http.StatusOK, "login.html", pVars)
}

func landingPage(c *gin.Context) {
	var pVars pageVars
	session := sessions.Default(c)
	pVars.Guest = session.Get("guest").(bool)
	pVars.PageTitle = "DMO Monitor and Management"

	c.HTML(http.StatusOK, "landing.html", pVars)
}

func wrapMiner(c *gin.Context) {
	var pVars pageVars
	session := sessions.Default(c)
	pVars.Guest = session.Get("guest").(bool)
	pVars.PageTitle = "DMO-Wrapminer"

	c.HTML(http.StatusOK, "wrapminer.html", pVars)
}

func statsPage(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	cloudKey := userIDList[userID].CloudKey

	var pVars pageVars
	pVars.Guest = session.Get("guest").(bool)

	mutex.Lock()
	for _, stats := range minerList[cloudKey] {
		if !stats.Late {
			pVars.Totalminers += 1
		}
	}

	pVars.PageTitle = "DMO Monitor - Statistics"
	pVars.NetHash = overallInfoTX[userID].NetHash

	pVars.CurrentPrice = currentPricePerDMO
	pVars.DollarsPerDay = currentPricePerDMO * overallInfoTX[userID].CurrentCoinsPerDay
	pVars.DollarsPerWeek = currentPricePerDMO * overallInfoTX[userID].CurrentCoinsPerDay * 7
	pVars.DollarsPerMonth = currentPricePerDMO * overallInfoTX[userID].CurrentCoinsPerDay * 30

	myMiners := make(map[string]mineRpc)
	for k, v := range minerList[cloudKey] {
		myMiners[k] = v
	}

	pVars.MinerList = myMiners
	pVars.Totalhash = totalHashG[cloudKey]
	pVars.WalletOverallStats = overallInfoTX[userID]
	pVars.WalletDailyStats = overallInfoTX[userID].DayStats
	pVars.WalletHourlyStats = overallInfoTX[userID].HourStats
	mutex.Unlock()

	pVars.AutoRefresh = myConfig.AutoRefreshSeconds
	pVars.DailyStatDays = myConfig.DailyStatDays

	pVars.Uptime = time.Since(progStartTime).Round(time.Second).String()

	c.HTML(http.StatusOK, "stats.html", pVars)

}
