package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Transaction struct {
	Address       string  `json:"address"`
	Category      string  `json:"category"`
	Amount        float64 `json:"amount"`
	Label         string  `json:"label"`
	Confirmations int64   `json:"confirmations"`
	Generated     bool    `json:"generated"`
	Blockhash     string  `json:"blockhash"`
	Blockheight   int64   `json:"blockheight"`
	Blockindex    int64   `json:"blockindex"`
	Blocktime     int64   `json:"blocktime"`
	TXID          string  `json:"txid"`
	dt            time.Time
	Time          int64 `json:"time"`
	TimeReceived  int64 `json:"timereceived"`
}

type OverallInfoTX struct {
	DailyAverage       float64
	HourlyAverage      float64
	WinPercent         float64
	Projection         string
	CurrentCoinsPerDay float64
}

var overallInfoTX OverallInfoTX

type DayStatTX struct {
	Day          string
	CoinCount    float64
	CoinsPerHour float64
	WinPercent   float64
}

var dayStatsTX []DayStatTX

type HourStatTX struct {
	Hour           int
	CoinCount      float64
	CoinsPerMinute float64
}

var hourStatsTX []HourStatTX

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
}

var addrStats AddrStatResponse

/*
http://dmo-monitor.com:9143/getminingstats
{
    "Addresses": "dy1qpfr5yhdkgs6jyuk945y23pskdxmy9ajefczsvm",
    "TZOffset": -28800
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

	myTime := time.Now()
	_, myTzOffset := myTime.Zone()

	var data = bytes.NewBufferString(`{"jsonrpc":"1.0","id":"curltest","Addresses":"` + c.AddrsToMonitor + `", "TZOffset": ` + strconv.Itoa(myTzOffset) + `, "NumDays": ` + strconv.Itoa(c.DailyStatDays) + `}`)
	req, err := http.NewRequest("GET", reqUrl.String(), data)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	mutex.Lock()
	if err := json.Unmarshal(bodyText, &addrStats); err != nil {
		mutex.Unlock()
		return
	}
	mutex.Unlock()

}
