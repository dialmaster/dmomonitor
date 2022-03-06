package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func doLogout(c *gin.Context) {
	session := sessions.Default(c)

	session.Set("dummy", "content") // this will mark the session as "written"
	session.Options(sessions.Options{MaxAge: -1})
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func removeLateMiner(c *gin.Context) {
	minerName := c.Query("minerName")
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	cloudKey := userIDList[userID].CloudKey

	mutex.Lock()
	if minerList[cloudKey][minerName].Late {
		delete(minerList[cloudKey], minerName)
	}
	mutex.Unlock()
}

func doUpdateTelegramID(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	telegramID := c.PostForm("telegram_id")
	log.Printf("Updating telegram id for user %d to %s\n", userID, telegramID)

	var formErrors []string

	valid := len(telegramID) >= 8
	if !valid {
		formErrors = append(formErrors, "Telegram ID must have minimum eight digits")
	}

	_, err := strconv.Atoi(telegramID)
	if err != nil {
		formErrors = append(formErrors, "Telegram ID must be numeric")
	}
	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		accountPage(c)
		return
	}

	_, err = db.Exec("UPDATE users SET telegram_user_id = ? WHERE ID = ?", telegramID, userID)

	if err != nil {
		formErrors = append(formErrors, "Sorry, unable to update telegram id at this time")
		log.Printf("Failed to update telegram id for user id %d, error: %s\n", userID, err.Error())
	} else {
		formErrors = append(formErrors, "Telegram user id update successful")
		getAllUserInfo()
	}

	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		accountPage(c)
	}

}

func doUpdateAddrs(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("ID").(int)
	addr := c.PostForm("addrs")
	var formErrors []string

	count := 0

	err := db.QueryRow("SELECT COUNT(*) FROM receiving_addresses where user_id = ?", userID).Scan(&count)
	if err != nil {
		formErrors = append(formErrors, "Failed to update receiving address")
		log.Printf("Failed to get count of receiving addresses in DB for user id %d: %s\n", userID, err.Error())
		c.Set("errors", formErrors)
		accountPage(c)
		return
	}

	if count > 0 {
		_, err = db.Exec("UPDATE receiving_addresses SET receiving_address = ? WHERE user_id = ?", addr, userID)
	} else {
		_, err = db.Exec("INSERT INTO receiving_addresses (user_id, receiving_address, display_name) values (?, ?, 'Default Address')", userID, addr)
	}

	if err != nil {
		formErrors = append(formErrors, "Failed to update receiving address")
		log.Printf("Failed to update receiving address in DB for user id %d: %s\n", userID, err.Error())
	} else {
		formErrors = append(formErrors, "Receiving address updated")
		getAllUserInfo()
		txStats()
	}
	c.Set("errors", formErrors)
	accountPage(c)

}

func doChangePass(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("ID").(int)

	oldpass := c.PostForm("psw")
	newpass := c.PostForm("newpsw")
	confirmnewpass := c.PostForm("confirmnewpsw")

	var formErrors []string

	if newpass != confirmnewpass {
		formErrors = append(formErrors, "New password must match new password confirmation")
	}

	valid := len(newpass) >= 8
	if !valid {
		formErrors = append(formErrors, "Password must have minimum eight characters")
	}

	if !CheckPasswordHash(oldpass, userIDList[userID].PasswordHash) {
		formErrors = append(formErrors, "You must correctly enter your old password to update your password")
	}

	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		accountPage(c)
		return
	}

	passHash, _ := HashPassword(newpass)

	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE ID = ?", passHash, userID)

	if err != nil {
		formErrors = append(formErrors, "Update password failed")
		log.Printf("Failed to update password for user id %d: %s\n", userID, err.Error())
	} else {
		formErrors = append(formErrors, "Password updated")
		getAllUserInfo()
	}

	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		accountPage(c)
	}

}

func doRegister(c *gin.Context) {
	username := c.PostForm("uname")
	password := c.PostForm("psw")

	var formErrors []string

	valid := len(password) >= 8
	if !valid {
		formErrors = append(formErrors, "Password must have minimum eight characters")
	}

	if _, ok := userList[username]; ok {
		formErrors = append(formErrors, "Please choose another username")
	}

	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		loginPage(c)
		return
	}

	cloudkey := createCloudKey()
	passHash, _ := HashPassword(password)

	res, err := db.Exec("INSERT INTO users (password_hash, username, created_at, cloud_key, last_active) values (?, ?, NOW(), ?, ?)", passHash, username, cloudkey, time.Now().Unix())

	if err != nil {
		formErrors = append(formErrors, "Registration failed")
		log.Printf("Registration insert failed for username %s: %s\n", username, err.Error())
		c.Set("errors", formErrors)
		loginPage(c)
		return
	}

	id, err := res.LastInsertId()

	if err != nil {
		formErrors = append(formErrors, "Registration failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}

	getAllUserInfo()

	session := sessions.Default(c)
	session.Set("ID", int(id))
	session.Set("guest", false)
	session.Save()

	c.Redirect(http.StatusFound, "/account")

}

func doLogin(c *gin.Context) {
	username := c.PostForm("uname")
	password := c.PostForm("psw")

	var formErrors []string

	if !CheckPasswordHash(password, userList[username].PasswordHash) {
		formErrors = append(formErrors, "Login failed")
		log.Printf("Login failed for user %s. Password hash did not match entered password\n", username)
		c.Set("errors", formErrors)
		loginPage(c)
		return
	}

	// Get a new cookie or set one if it does not exist:
	session := sessions.Default(c)
	id := userList[username].ID
	session.Set("ID", id)
	session.Set("guest", false)
	session.Save()

	updateUserLastActive(id)

	// Success! Set userid in session unset guest and redirect
	c.Redirect(http.StatusFound, "/")

}
