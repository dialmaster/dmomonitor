package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type mineRpc struct {
	Name           string
	Hashrate       int
	HashrateStr    string
	Accept         int
	Reject         int
	Submit         int
	Diff           float64
	LastReport     time.Time
	Late           bool
	MinerID        string
	HowLate        string
	Uptime         int
	UptimeDuration string
}

func getDMOWrapVersion(c *gin.Context) {
	type dmoWrapVersionInfo struct {
		Version string
	}
	var thisVersionInfo dmoWrapVersionInfo

	thisVersionInfo.Version = myConfig.DmoWrapVersionString

	c.JSON(200, thisVersionInfo)
}

func getMinerStatsRPC(c *gin.Context) {
	var thisStat mineRpc
	if err := c.BindJSON(&thisStat); err != nil {
		fmt.Printf("Got unhandled (bad) request!")
		return
	}

	thisStat.HashrateStr = formatHashNum(thisStat.Hashrate)
	thisStat.LastReport = time.Now()
	if thisStat.Uptime > 0 {
		someTime := time.Duration(thisStat.Uptime * 1000000000)
		secs := int(someTime.Seconds()) % 60
		mins := int(someTime.Minutes()) % 60
		hours := int(someTime.Hours()) % 24
		days := int(float64(someTime.Hours()) * float64(0.041666666))

		if mins == 0 && hours == 0 && days == 0 {
			thisStat.UptimeDuration = fmt.Sprintf("%ds", secs)
		} else if hours == 0 && days == 0 {
			thisStat.UptimeDuration = fmt.Sprintf("%dm%ds", mins, secs)
		} else if days == 0 {
			thisStat.UptimeDuration = fmt.Sprintf("%dh%dm", hours, mins)
		} else {
			thisStat.UptimeDuration = fmt.Sprintf("%dd%dh%dm", days, hours, mins)
		}

	} else {
		thisStat.UptimeDuration = "?"
	}

	cInterface, found := c.Get("cloudKey")
	cloudKey := ""
	if found {
		cloudKey = cInterface.(string)
	} else {
		return
	}
	if thisStat.MinerID == "" {
		thisStat.MinerID = thisStat.Name
	}

	log.Printf("Miner %s reported for user %s with hashrate %s\n", thisStat.Name, cloudKeyList[cloudKey].UserName, thisStat.HashrateStr)

	mutex.Lock()

	_, ok := minerList[cloudKey]
	if !ok {
		minerList[cloudKey] = make(map[string]mineRpc)
	}
	minerList[cloudKey][thisStat.MinerID] = thisStat

	mutex.Unlock()
}
