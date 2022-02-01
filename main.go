package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//TODO: Restore console output, code cleanup.

var versionString = "v1.2.0"

//go:embed templates/**
var tmplFS embed.FS

//go:embed js/**
var jsFS embed.FS

//go:embed img/**
var imgFS embed.FS

var c conf
var minerList = make(map[string]mineRpc)
var progStartTime = time.Now()
var mutex = &sync.Mutex{}
var totalHashG = ""
var walletStats = ""
var walletBalance = ""
var currentPricePerDMO = 0.0

func main() {

	c.getConf()
	if c.QuietMode {
		fmt.Printf("Starting monitor in quiet mode (no console output). Access stats at http://localhost:%s/stats\n", c.ServerPort)
	}

	if c.MinerLateTime < 20 {
		c.MinerLateTime = 20
	}
	if c.DailyStatDays < 2 {
		c.DailyStatDays = 3
	}
	// Don't let people get too nuts here
	if c.DailyStatDays > 21 {
		c.DailyStatDays = 21
	}
	if c.AutoRefreshSeconds < 10 && c.AutoRefreshSeconds > 0 {
		c.AutoRefreshSeconds = 10
	}

	if len(c.AddrsToMonitor) > 0 {
		txStats()
	}

	go func() {
		for {
			getCoinGeckoDMOPrice()
			if len(c.AddrsToMonitor) > 0 {
				txStats()
			}

			updateMinerStatus()
			if !c.QuietMode {
				consoleOutput()
			}
			time.Sleep(10 * time.Second)
		}
	}()
	handleRequests()
}

// https://api.coingecko.com/api/v3/simple/price?ids=dynamo-coin&vs_currencies=USD
func getCoinGeckoDMOPrice() {

	client := &http.Client{}
	reqUrl := url.URL{
		Scheme: "http",
		Host:   "api.coingecko.com",
		Path:   "api/v3/simple/price",
	}

	req, err := http.NewRequest("GET", reqUrl.String()+"?ids=dynamo-coin&vs_currencies=USD", nil)

	type geckoPrice struct {
		DynamoCoin struct {
			Usd float64 `json:"usd"`
		} `json:"dynamo-coin"`
	}

	resp, err := client.Do(req)
	// Sometimes the coingecko api call fails, and we do not want that to kill our app...
	if err != nil {
		return
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var myGeckoPrice geckoPrice

	if err := json.Unmarshal(bodyText, &myGeckoPrice); err != nil {
		return
	}

	currentPricePerDMO = myGeckoPrice.DynamoCoin.Usd
}

func sendOfflineNotificationToTelegram(minerName string) {
	params := url.Values{}
	params.Add("chat_id", c.TelegramUserId)
	params.Add("text", "Your miner '"+minerName+"' is offline")
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest("POST", "https://api.telegram.org/bot5084964646:AAEmnj-HIWsM1oBIHeCy03JsjBw_pG5I5Ik/sendMessage", body)

	// For now leaving errors unhandled... if the telegram notification fails it's not really a huge deal
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func statsPage(w http.ResponseWriter, r *http.Request) {
	fp := path.Join("templates", "stats.html")

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
	}

	var pVars pageVars

	for _, stats := range minerList {
		if !stats.Late {
			pVars.Totalminers += 1
		}
	}

	pVars.CurrentPrice = currentPricePerDMO
	pVars.DollarsPerDay = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay
	pVars.DollarsPerWeek = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay * 7
	pVars.DollarsPerMonth = currentPricePerDMO * overallInfoTX.CurrentCoinsPerDay * 30
	pVars.VersionString = versionString
	pVars.MinerList = minerList
	pVars.Totalhash = totalHashG
	pVars.WalletOverallStats = overallInfoTX
	pVars.WalletDailyStats = dayStatsTX
	pVars.WalletHourlyStats = hourStatsTX
	pVars.AutoRefresh = c.AutoRefreshSeconds
	pVars.DailyStatDays = c.DailyStatDays

	pVars.Uptime = time.Now().Sub(progStartTime).Round(time.Second).String()

	tmpl, err := template.ParseFS(tmplFS, fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, pVars); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

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

func getMinerStatsRPC(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var thisStat mineRpc
	err := decoder.Decode(&thisStat)
	if err != nil {
		panic(err)
	}

	thisStat.HashrateStr = formatHashNum(thisStat.Hashrate)
	thisStat.LastReport = time.Now()
	mutex.Lock()
	minerList[thisStat.Name] = thisStat
	mutex.Unlock()
}

func removeLateMiner(rw http.ResponseWriter, req *http.Request) {
	minerName := req.URL.Query().Get("minerName")
	// Do not allow removal of active miners
	if minerList[minerName].Late {
		delete(minerList, minerName)
	}
}

func handleRequests() {
	http.HandleFunc("/stats", statsPage)
	http.HandleFunc("/minerstats", getMinerStatsRPC)
	http.HandleFunc("/removeminer", removeLateMiner)
	http.Handle("/js/",
		http.StripPrefix("", http.FileServer(http.FS(jsFS))))

	http.Handle("/img/",
		http.StripPrefix("", http.FileServer(http.FS(imgFS))))

	log.Fatal(http.ListenAndServe(":"+c.ServerPort, nil))

}

func StringSpaced(text string, spacingchar string, numspaces int) string {
	numpads := numspaces - len(text)
	return text + strings.Repeat(spacingchar, numpads)
}

func formatHashNum(hashrate int) string {
	if hashrate < 1024 {
		return strconv.Itoa(hashrate)
	} else if hashrate > 1024 && hashrate < 1048576 {
		scaled := float64(hashrate) / float64(1024)

		return fmt.Sprintf("%.0fKH", scaled)
	} else if hashrate >= 1048576 && hashrate < 1073741824 {
		scaled := float64(hashrate) / float64(1048576)
		return fmt.Sprintf("%.2fMH", scaled)

	} else if hashrate >= 1073741824 && hashrate < 1099511627776 {
		scaled := float64(hashrate) / float64(1073741824)
		return fmt.Sprintf("%.2fGH", scaled)
	}

	return strconv.Itoa(hashrate) // Meh, it'll hopefully be a while before we exceed 999GH... if so just return the raw hash
}

func updateMinerStatus() {
	if len(minerList) > 0 {
		names := make([]string, 0)
		for name, _ := range minerList {
			names = append(names, name)
		}

		sort.Strings(names)

		totalHash := 0
		for _, name := range names {
			stats := minerList[name]
			howLong := time.Now().Sub(stats.LastReport).Round(time.Second)
			stats.HowLate = howLong.String()
			mutex.Lock()
			minerList[name] = stats
			mutex.Unlock()
			if howLong.Seconds() > c.MinerLateTime && stats.Late == false {
				stats.Late = true
				mutex.Lock()
				minerList[name] = stats
				mutex.Unlock()
				if len(c.TelegramUserId) > 0 {
					sendOfflineNotificationToTelegram(name)
				}
			} else if howLong.Seconds() <= c.MinerLateTime {
				stats.Late = false
				mutex.Lock()
				minerList[name] = stats
				mutex.Unlock()
				totalHash += stats.Hashrate
			}
		}
		totalHashG = formatHashNum(totalHash)
	}

	nDays := len(addrStats.DailyStats)
	overallInfoTX.CurrentCoinsPerDay = addrStats.DailyStats[nDays-2].Coins
	overallInfoTX.Projection = fmt.Sprintf("%.1f", addrStats.ProjectedCoinsToday)

	// Daily Average is the Average of the average of all days except today...
	tmpCoins := 0.0
	tmpPerc := 0.0
	dayStatsTX = nil
	for i := 0; i < nDays; i++ {

		var thisDay DayStatTX
		thisDay.Day = addrStats.DailyStats[i].Day
		thisDay.CoinCount = addrStats.DailyStats[i].Coins
		thisDay.WinPercent = addrStats.DailyStats[i].WinPercent
		thisDay.CoinsPerHour = thisDay.CoinCount / 24.0

		if i < (nDays - 1) {
			tmpCoins += addrStats.DailyStats[i].Coins
			tmpPerc += addrStats.DailyStats[i].WinPercent
		} else {
			thisDay.CoinsPerHour = addrStats.ProjectedCoinsToday / 24.0
		}

		dayStatsTX = append(dayStatsTX, thisDay)

	}
	overallInfoTX.DailyAverage = tmpCoins / (float64(nDays) - 1.0)
	overallInfoTX.HourlyAverage = overallInfoTX.DailyAverage / 24.0
	overallInfoTX.WinPercent = tmpPerc / (float64(nDays) - 1.0)

	nHours := len(addrStats.HourlyStats)

	hourStatsTX = nil
	for j := 0; j < nHours; j++ {
		var thisHour HourStatTX
		thisHour.Hour = addrStats.HourlyStats[j].Hour
		thisHour.CoinCount = float64(addrStats.HourlyStats[j].Coins)
		thisHour.CoinsPerMinute = float64(addrStats.HourlyStats[j].Coins) * (1.0 / 60.0)
		hourStatsTX = append(hourStatsTX, thisHour)
	}

}

// TODO: Add console output back in
func consoleOutput() {

	var H1Col = colorBrightCyan
	var H2Col = colorBrightGreen
	var H3Col = colorBrightBlue
	var WarnCol = colorRed

	fmt.Print("\033[H\033[2J") // Clear screen
	setColor(colorBrightWhite)
	fmt.Printf("\t\t\t\t\t\tDMO-Monitor %s\n\n", versionString)

	setColor(H1Col)
	fmt.Printf("\t\t\t\t\tMiner Monitoring Stats\n\n")

	setColor(H2Col)
	fmt.Printf("\t%s%s%s%s%s%s\n",
		StringSpaced("Miner Name", " ", 24),
		StringSpaced("Last Reported", " ", 35),
		StringSpaced("Hashrate", " ", 12),
		StringSpaced("Submitted", " ", 12),
		StringSpaced("Accepted", " ", 12),
		StringSpaced("Rejected", " ", 12))

	warnings := ""
	if len(minerList) > 0 {
		setColor(H3Col)
		names := make([]string, 0)
		for name, _ := range minerList {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			stats := minerList[name]
			setColor(H3Col)

			if stats.Late {
				warnings += "\n\tWARN: " + name + " has not reported in " + stats.HowLate + "\n"
				setColor(WarnCol)
			}

			fmt.Printf("\t%s%s%s%s%s%s\n",
				StringSpaced(name, " ", 24),
				StringSpaced(stats.LastReport.Format("2006-01-02 15:04:05"), " ", 35),
				StringSpaced(stats.HashrateStr, " ", 12),
				StringSpaced(strconv.Itoa(stats.Submit), " ", 12),
				StringSpaced(strconv.Itoa(stats.Accept), " ", 12),
				strconv.Itoa(stats.Reject),
			)
			setColor(H3Col)
		}

		fmt.Printf("\n\t%s%s\n",
			StringSpaced(fmt.Sprintf("Total Miners: %d", len(minerList)), " ", 32),
			StringSpaced(fmt.Sprintf("Total Hashrate: %s", totalHashG), " ", 32))
	} else {
		setColor(WarnCol)
		fmt.Printf("\t\t\t\tNo active miners\n")
	}

	if len(warnings) > 0 {
		setColor(WarnCol)
		fmt.Printf(warnings)
		setColor(H3Col)
	}

	if c.AddrsToMonitor != "" {

		setColor(H1Col)
		fmt.Printf("\n\n\t\t\t\t\tAddress Mining Stats for Address(es)\n\n")
		setColor(H2Col)
		fmt.Printf("\t%s%s%s%s\n",
			StringSpaced("Day", " ", 24),
			StringSpaced("Coins", " ", 35),
			StringSpaced("Coins/Hr", " ", 12),
			StringSpaced("Win Percent", " ", 12))
		setColor(H3Col)

		for _, day := range dayStatsTX {
			fmt.Printf("\t%s%s%s%s\n",
				StringSpaced(day.Day, " ", 24),
				StringSpaced(fmt.Sprintf("%.2f", day.CoinCount), " ", 35),
				StringSpaced(fmt.Sprintf("%.2f", day.CoinsPerHour), " ", 12),
				StringSpaced(fmt.Sprintf("%.2f", day.WinPercent), " ", 12))
		}
		fmt.Printf("\n\t%s\n",
			StringSpaced(fmt.Sprintf("Projected Coins Today: %s", overallInfoTX.Projection), " ", 40))

	} else {
		setColor(WarnCol)
		fmt.Printf("\n\n\t\t\t\t\tNo Receiving Address Statistics Available\n\n")
	}

}
