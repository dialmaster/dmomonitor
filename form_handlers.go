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

func doRegister(c *gin.Context) {
	username := c.PostForm("uname")
	password := c.PostForm("psw")

	var formErrors []string

	valid := len(password) >= 8
	if !valid {
		formErrors = append(formErrors, "Password must have minimum eight characters")
	}

	cnt := 0
	stmtOut, err := db.Prepare("SELECT count(*) FROM users WHERE username = ?")
	if err != nil {
		formErrors = append(formErrors, "Registration failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}
	stmtOut.QueryRow(username).Scan(&cnt)

	if cnt > 0 {
		formErrors = append(formErrors, "Please choose another username")
	}

	if len(formErrors) > 0 {
		c.Set("errors", formErrors)
		loginPage(c)
	}

	cloudkey := createCloudKey()
	passHash, _ := HashPassword(password)

	stmtIns, err := db.Prepare("INSERT INTO users (password_hash, username, created_at, cloud_key) values (?, ?, NOW(), ?)")

	if err != nil {
		formErrors = append(formErrors, "Registration failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}

	res, err := stmtIns.Exec(passHash, username, cloudkey)

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

	session := sessions.Default(c)
	session.Set("ID", id)
	session.Set("guest", false)
	session.Save()

	c.Redirect(http.StatusFound, "/")

}

func doLogin(c *gin.Context) {
	username := c.PostForm("uname")
	password := c.PostForm("psw")

	fmt.Printf("Posted form with username: %s and password %s\n", username, password)
	var formErrors []string

	var passHash string
	var id int
	stmtOut, err := db.Prepare("SELECT id, password_hash FROM users WHERE username = ?")
	if err != nil {
		formErrors = append(formErrors, "Login failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}
	stmtOut.QueryRow(username).Scan(&id, &passHash)

	if !CheckPasswordHash(password, passHash) {
		formErrors = append(formErrors, "Login failed")
		c.Set("errors", formErrors)
		loginPage(c)
	}

	// Get a new cookie or set one if it does not exist:
	session := sessions.Default(c)
	session.Set("ID", id)
	session.Set("guest", false)
	session.Save()

	fmt.Printf("Login success! User id is: %d\n", id)

	// Success! Set userid in session unset guest and redirect
	c.Redirect(http.StatusFound, "/")

}
