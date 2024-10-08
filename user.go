package main

import (
	"log"
	"time"
)

type UserData struct {
	PasswordHash       string
	UserName           string
	CloudKey           string
	ID                 int
	ReceivingAddresses []ReceivingAddress
	TelegramUserId     string
	Paid               int
	Admin              int
	LastActive         int64 // unix_epoch
	TimeZone           string
}

type ReceivingAddress struct {
	DisplayName      string
	ReceivingAddress string
}

var lastActive = make(map[int]int64)

var userList = make(map[string]UserData)

var cloudKeyList = make(map[string]UserData)

var userIDList = make(map[int]UserData)

func updateUserLastActive(userID int) {
	_, err := db.Exec("UPDATE users SET last_active = ? WHERE ID = ?", time.Now().Unix(), userID)

	if err != nil {
		log.Printf("Failed to update last active time for user id %d: %s\n", userID, err.Error())
	}

}

// Get all users in the DB:
func getAllUserInfo() {
	results, err := db.Query("select password_hash, username, cloud_key, id, telegram_user_id, paid, admin, last_active, timezone from users")

	if err != nil {
		log.Printf("Unable to get user data from DB\n")
		return
	}

	for results.Next() {
		var thisUser UserData
		err = results.Scan(&thisUser.PasswordHash, &thisUser.UserName, &thisUser.CloudKey, &thisUser.ID, &thisUser.TelegramUserId, &thisUser.Paid, &thisUser.Admin, &thisUser.LastActive, &thisUser.TimeZone)
		if err != nil {
			log.Printf("Unable to read user from DB\n")
			continue
		}
		results2, err2 := db.Query("select display_name, receiving_address from receiving_addresses where user_id = ?", thisUser.ID)

		if err2 != nil {
			log.Printf("Unable to get receiving addresses for user id: %d\n", thisUser.ID)
		} else {
			var thisAddress ReceivingAddress
			for results2.Next() {
				err = results2.Scan(&thisAddress.DisplayName, &thisAddress.ReceivingAddress)
				if err != nil {
					log.Printf("Unable to read receiving address from DB for user_id %d\n", thisUser.ID)
				} else {
					log.Printf("Got receiving address %s for user %s", thisAddress.ReceivingAddress, thisUser.UserName)
					thisUser.ReceivingAddresses = append(thisUser.ReceivingAddresses, thisAddress)
				}
			}
		}

		mutex.Lock()
		userList[thisUser.UserName] = thisUser
		cloudKeyList[thisUser.CloudKey] = thisUser
		userIDList[thisUser.ID] = thisUser
		lastActive[thisUser.ID] = thisUser.LastActive
		mutex.Unlock()
	}

}
