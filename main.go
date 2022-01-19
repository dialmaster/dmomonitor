package main

import (
	"bytes"
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

var c conf
var minerList = make(map[string]mineRpc)
var progStartTime = time.Now()
var mutex = &sync.Mutex{}

var totalHashG = ""
var walletStats = ""
var walletBalance = ""

func main() {

	c.getConf()

	go func() {
		for {
			consoleOutput()
			time.Sleep(6 * time.Second)
		}
	}()
	handleRequests()
}

func getWalletsBalance(walletNames string) string {
	walletBalanceTotal := 0.0000

	type walletResp struct {
		Balance float64 `json:"result"`
		ID      string  `json:"id"`
	}

	var wallets = strings.Split(walletNames, ",")

	for _, w := range wallets {
		var thisWallet = strings.TrimSpace(w)

		client := &http.Client{}
		reqUrl := url.URL{
			Scheme: "http",
			Host:   c.NodeIP + ":" + c.NodePort,
			Path:   "wallet/" + thisWallet,
		}

		var data = bytes.NewBufferString(`{"jsonrpc":"1.0","id":"curltest","method":"getbalance"}`)

		req, err := http.NewRequest("POST", reqUrl.String(), data)
		req.SetBasicAuth(c.NodeUser, c.NodePass)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
			return "Unable to make request to wallet: " + err.Error()
		}
		bodyText, err := io.ReadAll(resp.Body)

		var myWalletBalance walletResp

		if err := json.Unmarshal(bodyText, &myWalletBalance); err != nil {
			return "Unable to decode json from wallet request: " + err.Error()
		}

		walletBalanceTotal += myWalletBalance.Balance
	}

	return fmt.Sprintf("%.3f", walletBalanceTotal)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fp := path.Join("templates", "index.html")

	type pageVars struct {
		Uptime             string
		MinerList          map[string]mineRpc
		Totalhash          string
		Totalminers        string
		Walletbalance      string
		WalletOverallStats OverallInfoTX
		WalletDailyStats   []DayStatTX
		WalletHourlyStats  []HourStatTX
	}

	var pVars pageVars

	pVars.MinerList = minerList
	pVars.Totalhash = totalHashG
	pVars.Walletbalance = walletBalance
	pVars.WalletOverallStats = overallInfoTX
	pVars.WalletDailyStats = dayStatsTX
	pVars.WalletHourlyStats = hourStatsTX

	upTime := time.Now().Sub(progStartTime).Round(time.Second)
	pVars.Uptime = upTime.String()

	tmpl, err := template.ParseFiles(fp)
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
	delete(minerList, req.URL.Query().Get("minerName"))
}

func handleRequests() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/minerstats", getMinerStatsRPC)
	http.HandleFunc("/removeminer", removeLateMiner)
	http.Handle("/js/",
		http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))

	log.Fatal(http.ListenAndServe(":"+c.ServerPort, nil))
}

func StringSpaced(text string, spacingchar string, numspaces int) string {
	numpads := numspaces - len(text)
	return text + strings.Repeat(spacingchar, numpads)
}

func formatHashNum(hashrate int) string {
	if hashrate < 1024 {
		return strconv.Itoa(hashrate)
	} else if hashrate > 1024 && hashrate < 10000000 {
		return strconv.Itoa(hashrate/1024) + "KH"
	} else if hashrate >= 10000000 && hashrate < 1000000000 {
		return strconv.Itoa(hashrate/1048576) + "MH"
	} else if hashrate >= 1000000000 {
		return strconv.Itoa(hashrate/1073741824) + "GH"
	}
	return "ERR"
}

func consoleOutput() {

	if len(c.WalletsToMonitor) > 0 && c.NodeIP != "XXX.XXX.XXX.XXX" {
		walletStats = txStats()
		walletBalance = getWalletsBalance(c.WalletsToMonitor)
	}

	fmt.Print("\033[H\033[2J") // Clear screen
	setColor(colorWhite)
	fmt.Printf("\t\t\t\tDMO Mining Monitor\n\n")

	setColor(colorYellow)
	fmt.Printf("\t%s%s%s%s%s%s\n",
		StringSpaced("Miner Name", " ", 24),
		StringSpaced("Last Reported", " ", 35),
		StringSpaced("Hashrate", " ", 12),
		StringSpaced("Submitted", " ", 12),
		StringSpaced("Accepted", " ", 12),
		StringSpaced("Rejected", " ", 12))

	warnings := ""
	if len(minerList) > 0 {
		setColor(colorGreen)
		names := make([]string, 0)
		for name, _ := range minerList {
			names = append(names, name)
		}

		sort.Strings(names)

		totalHash := 0
		for _, name := range names {
			stats := minerList[name]
			howLong := time.Now().Sub(stats.LastReport)
			howLong -= howLong % time.Second
			if howLong.Seconds() > 12 {
				setColor(colorRed)
				stats.Late = true
				mutex.Lock()
				minerList[name] = stats
				mutex.Unlock()
				warnings += "\n\tWARN: " + name + " has not reported in " + howLong.String() + "\n"
			} else {
				stats.Late = false
				totalHash += stats.Hashrate
				setColor(colorGreen)
			}

			fmt.Printf("\t%s%s%s%s%s%s\n",
				StringSpaced(name, " ", 24),
				StringSpaced(stats.LastReport.Format("2006-01-02 15:04:05"), " ", 35),
				StringSpaced(stats.HashrateStr, " ", 12),
				StringSpaced(strconv.Itoa(stats.Submit), " ", 12),
				StringSpaced(strconv.Itoa(stats.Accept), " ", 12),
				strconv.Itoa(stats.Reject),
			)
			setColor(colorGreen)
		}

		fmt.Printf("\n\tTotal Miners: %d", len(minerList))
		totalHashG = formatHashNum(totalHash)
		fmt.Printf("\n\tTotal Hashrate: %s", totalHashG)
	} else {
		setColor(colorRed)
		fmt.Printf("\t\t\t\tNo active miners\n")
	}

	if len(warnings) > 0 {
		setColor(colorRed)
		fmt.Printf(warnings)
		setColor(colorGreen)
	}

	if c.WalletsToMonitor != "MyExampleWalletName1,MyExampleWalletName2" {
		fmt.Printf("\n\tWallets Combined Balance (%s): %s\n", c.WalletsToMonitor, walletBalance)

		fmt.Printf("\n\n\n")
		setColor(colorYellow)
		fmt.Printf("\t\t\t\tWallet Mining Stats for Wallets: %s\n", c.WalletsToMonitor)
		setColor(colorGreen)

		fmt.Println(walletStats)
	}
}
