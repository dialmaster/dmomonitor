package main

import (
	"fmt"
	"net/http"

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

	passHash, _ := HashPassword(newpass)

	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE ID = ?", passHash, userID)

	if err != nil {
		formErrors = append(formErrors, "Update password failed")
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
	}

	cloudkey := createCloudKey()
	passHash, _ := HashPassword(password)

	res, err := db.Exec("INSERT INTO users (password_hash, username, created_at, cloud_key) values (?, ?, NOW(), ?)", passHash, username, cloudkey)

	if err != nil {
		formErrors = append(formErrors, "Registration failed")
		c.Set("errors", formErrors)
		loginPage(c)
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

	c.Redirect(http.StatusFound, "/")

}

func doLogin(c *gin.Context) {
	username := c.PostForm("uname")
	password := c.PostForm("psw")

	fmt.Printf("Posted form with username: %s and password %s\n", username, password)
	var formErrors []string

	if !CheckPasswordHash(password, userList[username].PasswordHash) {
		formErrors = append(formErrors, "Login failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}

	// Get a new cookie or set one if it does not exist:
	session := sessions.Default(c)
	id := userList[username].ID
	session.Set("ID", id)
	session.Set("guest", false)
	session.Save()

	// Success! Set userid in session unset guest and redirect
	c.Redirect(http.StatusFound, "/")

}
