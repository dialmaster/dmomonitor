package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	//	"strconv"
	"strings"
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

type StatData struct {
	firstBlock int64
	lastBlock  int64
	duration   time.Duration
	blocks     int64
	coins      float64
}

type OverallInfoTX struct {
	DailyAverage  float64
	HourlyAverage float64
	WinPercent    float64
	Projection    string
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

func (s *StatData) record(tx *Transaction) {
	if s.firstBlock == 0 || tx.Blockheight < s.firstBlock {
		s.firstBlock = tx.Blockheight
	}
	if tx.Blockheight > s.lastBlock {
		s.lastBlock = tx.Blockheight
	}
	s.coins += tx.Amount
	s.blocks++
}

func (s *StatData) roughPercent() float64 {
	if s.blocks == 0 {
		return 0
	}
	return 100.0 * (float64(s.blocks) / float64(s.lastBlock-s.firstBlock))
}

func getDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func fetchTX(u *url.URL) []*Transaction {
	var resp struct {
		Results []*Transaction `json:"result"`
	}

	var data = bytes.NewBufferString(`{"jsonrpc":"1.0","id":"curltest","method":"listtransactions","params":["*", 10000, 0]}`)
	var err = doPost(u, data, &resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to POST to URL %q: %s", u.String(), err)
		os.Exit(2)
	}

	return resp.Results
}

func doPost(u *url.URL, data io.Reader, resp interface{}) error {
	req, err := http.NewRequest("POST", u.String(), data)
	req.SetBasicAuth(c.NodeUser, c.NodePass)

	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Content-Type", "text/plain")

	cli := &http.Client{}
	r, err := cli.Do(req)

	if err != nil {
		return err
	}
	var body []byte
	body, err = io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	return nil
}

func txStats() string {
	var outString = ""
	var urlString = "http://" + c.NodeIP + ":" + c.NodePort
	var reportDays = c.DailyStatDays
	var wallets = strings.Split(c.WalletsToMonitor, ",")

	var u, err = url.Parse(urlString)
	if err != nil {
		fmt.Printf("ERR")
	}
	var txList []*Transaction
	for _, w := range wallets {
		u.Path = "/wallet/" + w
		txList = append(txList, fetchTX(u)...)
	}

	var reportStats StatData
	var dailyStats = make([]StatData, reportDays)
	var hourlyStats = make([]StatData, 24)
	var now = time.Now()
	var nowDay = getDay(now)
	var daysAgo = time.Duration(reportDays-1) * time.Hour * -24
	var beginReport = nowDay.Add(daysAgo)

	for _, tx := range txList {
		tx.dt = time.Unix(tx.TimeReceived, 0)

		if !tx.Generated {
			continue
		}
		if tx.Confirmations < 6 {
			continue
		}

		if tx.dt.Before(beginReport) {
			continue
		}

		reportStats.record(tx)

		var dayIndex = int(tx.dt.Sub(beginReport) / time.Hour / 24)
		dailyStats[dayIndex].record(tx)

		if dayIndex == reportDays-1 {
			hourlyStats[tx.dt.Hour()].record(tx)
		}
	}

	var total = reportStats.coins
	mutex.Lock()
	overallInfoTX.DailyAverage = total / float64(reportDays) // TODO: Fix this! It is wrong because it includes today, which is possibly just starting
	overallInfoTX.HourlyAverage = total / float64(reportDays) / 24.0
	overallInfoTX.WinPercent = reportStats.roughPercent()
	mutex.Unlock()
	outString += fmt.Sprintf("\tDaily average: %0.2f\n", total/float64(reportDays))
	outString += fmt.Sprintf("\tHourly average: %0.2f\n", total/float64(reportDays)/24.0)
	outString += fmt.Sprintf("\tRough Block Win Percent: %0.4f%%\n", reportStats.roughPercent())

	mutex.Lock()
	dayStatsTX = dayStatsTX[:0]
	mutex.Unlock()
	for i := 0; i < reportDays; i++ {
		var dayStat DayStatTX
		var projection = ""
		var coins = dailyStats[i].coins
		var hours = 24.0
		var when = beginReport.Add(time.Hour * 24 * time.Duration(i)).Format("2006-01-02")
		dayStat.Day = when
		dayStat.CoinCount = coins
		dayStat.WinPercent = dailyStats[i].roughPercent()
		if i == reportDays-1 {
			dayStat.Day = "Today"
			hours = float64(now.Hour()) + float64(now.Minute())/60.0
			projection = fmt.Sprintf(" (~ %0.2f expected)", coins/hours*24)
			overallInfoTX.Projection = fmt.Sprintf("%0.2f", coins/hours*24)
		}
		dayStat.CoinsPerHour = coins / hours
		mutex.Lock()
		dayStatsTX = append(dayStatsTX, dayStat)
		mutex.Unlock()
		outString += fmt.Sprintf("\t%s:\t\t\t%8.2f\t\t%0.2f/h\t\tWin%%: %0.4f%%%s\n", when, coins, coins/hours, dailyStats[i].roughPercent(), projection)
	}

	mutex.Lock()
	hourStatsTX = hourStatsTX[:0]
	mutex.Unlock()
	for i := 0; i <= now.Hour(); i++ {
		var hourStat HourStatTX
		var projection = ""
		var coins = hourlyStats[i].coins
		var minutes = 60.0
		var when = fmt.Sprintf("%s hour %02d", getDay(now).Format("2006-01-02"), i)
		hourStat.Hour = i
		hourStat.CoinCount = coins

		if i == now.Hour() {
			minutes = float64(now.Minute()) + float64(now.Second())/60
			projection = fmt.Sprintf("(~ %0.2f expected)", coins/minutes*60)
		}
		hourStat.CoinsPerMinute = coins / minutes
		mutex.Lock()
		hourStatsTX = append(hourStatsTX, hourStat)
		mutex.Unlock()

		outString += fmt.Sprintf("\t    %s:\t\t%8.2f\t\t%0.2f/m\t%s\n", when, coins, coins/minutes, projection)
	}
	return outString
}
