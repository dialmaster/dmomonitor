package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type OverallInfoTX struct {
	DailyAverage       float64
	HourlyAverage      float64
	WinPercent         float64
	Projection         string
	CurrentCoinsPerDay float64
	NetHash            string
	DayStats           []DayStatTX
	HourStats          []HourStatTX
	TotalActiveMiners  int
}

var overallInfoTX = make(map[int]OverallInfoTX)

type DayStatTX struct {
	Day          string
	CoinCount    float64
	CoinsPerHour float64
	WinPercent   float64
}

type HourStatTX struct {
	Hour           int
	CoinCount      float64
	CoinsPerMinute float64
}

type AddrStatResponse struct {
	HourlyStats []struct {
		Hour       int     `json:"Hour"`
		Coins      float64 `json:"Coins"`
		ChainCoins float64 `json:"ChainCoins"`
		WinPercent float64 `json:"WinPercent"`
	} `json:"HourlyStats"`
	DailyStats []struct {
		Day        string  `json:"Day"`
		Coins      float64 `json:"Coins"`
		ChainCoins float64 `json:"ChainCoins"`
		WinPercent float64 `json:"WinPercent"`
	} `json:"DailyStats"`
	ProjectedCoinsToday float64 `json:"ProjectedCoinsToday"`
	NetHash             float64 `json:"NetHash"`
}

// key is user id
var addrStats = make(map[int]AddrStatResponse)

/*
http://dmo-monitor.com:9143/getminingstats
{
    "Addresses": "dy1qpfr5yhdkgs6jyuk945y23pskdxmy9ajefczsvm",
}
NOTE: If an error occurs, then just do not update the stats....
*/
func txStats() {
	client := &http.Client{}
	reqUrl := url.URL{
		Scheme: "http",
		Host:   "dmo-monitor.com:9143",
		Path:   "getminingstats",
	}

	for userID, addresses := range userIDList {
		// Not really an error... some users may not have configured this.
		if len(addresses.ReceivingAddresses) == 0 {
			continue
		}

		var addrsToMonitor = ""
		for _, address := range addresses.ReceivingAddresses {
			addrsToMonitor += address.ReceivingAddress + ","
		}
		addrsToMonitor = addrsToMonitor[:len(addrsToMonitor)-1]

		var thisAddrStat AddrStatResponse

		var data = bytes.NewBufferString(`{"jsonrpc":"1.0","id":"curltest","Addresses":"` + addrsToMonitor + `", "NumDays": ` + strconv.Itoa(myConfig.DailyStatDays) + `}`)
		req, err := http.NewRequest("GET", reqUrl.String(), data)
		if err != nil {
			log.Printf("Unable to make request to dmo-statservice: %s", err.Error())
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Unable to make request to dmo-statservice: %s", err.Error())
			continue
		}
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Unable to make request to dmo-statservice: %s", err.Error())
			continue
		}

		if err := json.Unmarshal(bodyText, &thisAddrStat); err != nil {
			log.Printf("Unable to make request to dmo-statservice: %s", err.Error())
			continue
		}

		mutex.Lock()
		addrStats[userID] = thisAddrStat
		mutex.Unlock()
	}

}
