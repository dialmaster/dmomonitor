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
	HowLate     string
}

func getMinerStatsRPC(c *gin.Context) {
	var thisStat mineRpc
	if err := c.BindJSON(&thisStat); err != nil {
		fmt.Printf("Got unhandled (bad) request!")
		return
	}

	thisStat.HashrateStr = formatHashNum(thisStat.Hashrate)
	thisStat.LastReport = time.Now()
	mutex.Lock()
	minerList[thisStat.Name] = thisStat
	mutex.Unlock()
}

func removeLateMiner(c *gin.Context) {
	minerName := c.Query("minerName")
	// Do not allow removal of active miners
	if minerList[minerName].Late {
		delete(minerList, minerName)
	}
}
