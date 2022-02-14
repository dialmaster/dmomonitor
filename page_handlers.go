package main

import (
	"fmt"
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
}

func accountPage(c *gin.Context) {
	var pVars pageVars
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	pVars.PageTitle = "DMO Monitor and Management"
	pVars.UserName = userIDList[userID].UserName
	pVars.CloudKey = userIDList[userID].CloudKey
	errInterface, found := c.Get("errors")
	if found {
		pVars.Errors = errInterface.([]string)
	}

	c.HTML(http.StatusOK, "account.html", pVars)
}

func loginPage(c *gin.Context) {
	session := sessions.Default(c)
	var pVars pageVars
	pVars.PageTitle = "DMO Monitor and Management"
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

	fmt.Printf("landingPage: count %s!", c.GetString("count_val"))
	c.HTML(http.StatusOK, "landing.html", pVars)
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
	mutex.Unlock()

	pVars.PageTitle = "DMO Monitor"
	pVars.NetHash = overallInfoTX.NetHash

	pVars.CurrentPrice = currentPricePerDMO
	pVars.DollarsPerDay = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay
	pVars.DollarsPerWeek = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay * 7
	pVars.DollarsPerMonth = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay * 30
	pVars.VersionString = versionString

	myMiners := make(map[string]mineRpc)
	mutex.Lock()
	for k, v := range minerList[cloudKey] {
		myMiners[k] = v
	}
	mutex.Unlock()

	pVars.MinerList = myMiners
	pVars.Totalhash = totalHashG[cloudKey]
	pVars.WalletOverallStats = overallInfoTX
	pVars.WalletDailyStats = dayStatsTX
	pVars.WalletHourlyStats = hourStatsTX
	pVars.AutoRefresh = myConfig.AutoRefreshSeconds
	pVars.DailyStatDays = myConfig.DailyStatDays

	pVars.Uptime = time.Since(progStartTime).Round(time.Second).String()

	c.HTML(http.StatusOK, "stats.html", pVars)

}
