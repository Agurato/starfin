package server

import (
	"net/http"
	"strings"

	"github.com/matthewhartstonge/argon2"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// HandlePOSTRegister handles registration from POST request
func HandlePOSTRegister(c *gin.Context) {
	// Fetch username and passwords from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")
	// isAdmin := strings.Trim(c.PostForm("isAdmin"), " ")

	if err := AddUser(username, password1, password2, false); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/register.html", gin.H{
			"title":    "Register",
			"error":    err.Error(),
			"username": username,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

// HandlePOSTLogin handles login from POST request
func HandlePOSTLogin(c *gin.Context) {
	session := sessions.Default(c)
	userColl := mongoDb.Collection("users")
	// Fetch username and password from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password := strings.Trim(c.PostForm("password"), " ")

	// Check username length
	if len(username) < 3 || len(username) > 25 {
		RenderHTML(c, http.StatusUnauthorized, "pages/login.html", gin.H{
			"title":    "Login",
			"error":    "Username must be between 3 and 25 characters",
			"username": username,
		})
		return
	}

	// Fetch encoded password from DB
	var user User
	if err := userColl.FindOne(MongoCtx, bson.M{"name": username}).Decode(&user); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/login.html", gin.H{
			"title":    "Login",
			"error":    "Authentication failed",
			"username": username,
		})
		return
	}

	// Check if the username/password combination is valid
	ok, err := argon2.VerifyEncoded([]byte(password), []byte(user.Password))
	if err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/login.html", gin.H{
			"title":    "Login",
			"error":    "An error occured while logging you in",
			"username": username,
		})
		return
	}
	if !ok {
		RenderHTML(c, http.StatusUnauthorized, "pages/login.html", gin.H{
			"title":    "Login",
			"error":    "Authentication failed",
			"username": username,
		})
		return
	}

	// Save cookie
	userSession := User{
		ID:   user.ID,
		Name: user.Name,
	}
	session.Set(UserKey, userSession)
	if err := session.Save(); err != nil {
		RenderHTML(c, http.StatusInternalServerError, "pages/login.html", gin.H{
			"title":    "Login",
			"error":    "Server had trouble to log you in",
			"username": username,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

// HandlePOSTSetPassword handles changing password from POST request
func HandlePOSTSetPassword(c *gin.Context) {
	session := sessions.Default(c)
	userColl := mongoDb.Collection("users")
	user := session.Get(UserKey).(User)
	argon := argon2.DefaultConfig()
	// Fetch username and password from POST data
	oldPassword := strings.Trim(c.PostForm("old-password"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	// Check new passwords match
	if password1 != password2 {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "New passwords don't match",
		})
		return
	}

	// Check password length
	if len(password1) < 8 {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "Passwords must be at least 8 characters long",
		})
	}

	// Fetch encoded password from DB
	var userDB User
	if err := userColl.FindOne(MongoCtx, bson.M{"name": user.Name}).Decode(&userDB); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "An error occured while checking for your password",
		})
		return
	}

	// Check if the username/password combination is valid
	ok, err := argon2.VerifyEncoded([]byte(oldPassword), []byte(userDB.Password))
	if err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "An error occured while checking for your password",
		})
		return
	}
	if !ok {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "Authentication failed",
		})
		return
	}

	// Hash & encode password
	encoded, err := argon.HashEncoded([]byte(password1))
	if err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "An error occured while saving your password",
		})
		return
	}

	change := bson.M{"$set": bson.M{"password": string(encoded)}}
	if _, err := userColl.UpdateOne(MongoCtx, bson.M{"_id": userDB.ID}, change); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "An error occured while saving your password",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?setpassword=success")
}
