package main

import (
	"log"
	"net/http"
	"sort"
	"time"

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
	UserID             int
	Admin              int
	Paid               int
}

func getContextpVars(c *gin.Context) pageVars {
	pVars, found := c.Get("pVars")
	if !found {
		log.Printf("Unable to get pVars for context. Shouldn't be possible!\n")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to get context for page"})
	}
	return pVars.(pageVars)
}

func accountPage(c *gin.Context) {
	pVars := getContextpVars(c)
	pVars.PageTitle = "DMO Monitor - My Account"

	errInterface, found := c.Get("errors")
	if found {
		pVars.Errors = errInterface.([]string)
	}

	pVars.Addresses = ""
	for _, address := range userIDList[pVars.UserID].ReceivingAddresses {
		pVars.Addresses += address.ReceivingAddress + ","
	}
	if len(pVars.Addresses) > 1 {
		pVars.Addresses = pVars.Addresses[:len(pVars.Addresses)-1]
	}

	c.HTML(http.StatusOK, "account.html", pVars)
}

func loginPage(c *gin.Context) {
	pVars := getContextpVars(c)
	pVars.PageTitle = "DMO Monitor - Login"

	errInterface, found := c.Get("errors")
	if found {
		pVars.Errors = errInterface.([]string)
	}

	c.HTML(http.StatusOK, "login.html", pVars)
}

func landingPage(c *gin.Context) {
	pVars := getContextpVars(c)
	pVars.PageTitle = "DMO Monitor and Management"

	c.HTML(http.StatusOK, "landing.html", pVars)
}

func wrapMiner(c *gin.Context) {
	pVars := getContextpVars(c)
	pVars.PageTitle = "DMO-Wrapminer"

	c.HTML(http.StatusOK, "wrapminer.html", pVars)
}

func adminPage(c *gin.Context) {
	pVars := getContextpVars(c)
	if pVars.Admin != 1 {
		c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	type adminViewUser struct {
		UserName            string
		LastActive          string
		ProjectedCoinsToday string
		CurrentHash         string
		Admin               int
		Paid                int
		TotalActiveMiners   int
		ID                  int
	}

	type adminPVars struct {
		AutoRefresh int
		Uptime      int
		UserList    []adminViewUser
		PageTitle   string
		Guest       bool
		Admin       int
	}

	var myPVars adminPVars

	myPVars.PageTitle = "DMO Monitor - Admin"
	myPVars.Guest = pVars.Guest
	myPVars.Admin = pVars.Admin

	mutex.Lock()
	for id, user := range userIDList {
		var thisAVUser adminViewUser
		thisAVUser.ID = user.ID
		thisAVUser.UserName = user.UserName
		thisAVUser.LastActive = lastActive[id]
		thisAVUser.Admin = user.Admin
		thisAVUser.Paid = user.Paid
		thisAVUser.TotalActiveMiners = overallInfoTX[id].TotalActiveMiners
		thisAVUser.ProjectedCoinsToday = overallInfoTX[id].Projection

		thisAVUser.CurrentHash = totalHashG[user.CloudKey]
		myPVars.UserList = append(myPVars.UserList, thisAVUser)
	}

	sort.Slice(myPVars.UserList, func(i, j int) bool {
		return myPVars.UserList[i].ID < myPVars.UserList[j].ID
	})

	mutex.Unlock()

	c.HTML(http.StatusOK, "admin.html", myPVars)
}

func statsPage(c *gin.Context) {
	pVars := getContextpVars(c)
	pVars.PageTitle = "DMO Monitor - Statistics"

	userID := pVars.UserID
	cloudKey := userIDList[userID].CloudKey

	mutex.Lock()
	pVars.Totalminers = overallInfoTX[userID].TotalActiveMiners
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
