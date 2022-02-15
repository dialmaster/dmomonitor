package main

import "log"

type UserData struct {
	PasswordHash       string
	UserName           string
	CloudKey           string
	ID                 int
	ReceivingAddresses []ReceivingAddress
}

type ReceivingAddress struct {
	DisplayName      string
	ReceivingAddress string
}

var userList = make(map[string]UserData)

var cloudKeyList = make(map[string]UserData)

var userIDList = make(map[int]UserData)

// Get all users in the DB:
func getAllUserInfo() {
	results, err := db.Query("select password_hash, username, cloud_key, id from users")

	if err != nil {
		log.Printf("Unable to get user data from DB\n")
		return
	}

	for results.Next() {
		var thisUser UserData
		err = results.Scan(&thisUser.PasswordHash, &thisUser.UserName, &thisUser.CloudKey, &thisUser.ID)
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
		mutex.Unlock()
	}

}
