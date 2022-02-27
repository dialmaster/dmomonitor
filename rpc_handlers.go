package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type mineRpc struct {
	Name        string
	Hashrate    int
	HashrateStr string
	Accept      int
	Reject      int
	Submit      int
	LastReport  time.Time
	Late        bool
	MinerID     string
	HowLate     string
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

	mutex.Lock()

	_, ok := minerList[cloudKey]
	if !ok {
		minerList[cloudKey] = make(map[string]mineRpc)
	}
	minerList[cloudKey][thisStat.MinerID] = thisStat

	mutex.Unlock()
}
