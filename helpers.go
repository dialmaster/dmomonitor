package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserData struct {
	PasswordHash string
	UserName     string
	CloudKey     string
	ID           int
}

var userList = make(map[string]UserData)

var cloudKeyList = make(map[string]UserData)

var userIDList = make(map[int]UserData)

// Get all users in the DB:
func getAllUserInfo() {
	results, err := db.Query("select password_hash, username, cloud_key, id from users")

	if err != nil {
		panic(err.Error())
	}

	for results.Next() {
		var thisUser UserData
		err = results.Scan(&thisUser.PasswordHash, &thisUser.UserName, &thisUser.CloudKey, &thisUser.ID)
		if err != nil {
			panic(err.Error())
		}

		mutex.Lock()
		userList[thisUser.UserName] = thisUser
		cloudKeyList[thisUser.CloudKey] = thisUser
		userIDList[thisUser.ID] = thisUser

		mutex.Unlock()
	}

}

func createCloudKey() string {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Unable to generate random number for cloud key")
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
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
	params.Add("chat_id", myConfig.TelegramUserId)
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
