package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Agurato/down-low-d/internal/media"
	"github.com/matthewhartstonge/argon2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// HandlePOSTStart handles registration (only available for first account)
func HandlePOSTStart(c *gin.Context) {
	if GetUserNb() != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin user has already been created"})
		return
	}
	// Fetch username and passwords from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	if err := AddUser(username, password1, password2, true); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/start.html", gin.H{
			"title":    "Start",
			"error":    err.Error(),
			"username": username,
		})
		return
	}

	setupDone = true
	c.Redirect(http.StatusSeeOther, "/admin")
}

// HandlePOSTLogin handles login from POST request
func HandlePOSTLogin(c *gin.Context) {
	session := sessions.Default(c)
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
	if err := mongoUsers.FindOne(MongoCtx, bson.M{"name": username}).Decode(&user); err != nil {
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
	user.Password = ""
	session.Set(UserKey, user)
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
	if err := mongoUsers.FindOne(MongoCtx, bson.M{"name": user.Name}).Decode(&userDB); err != nil {
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
	if _, err := mongoUsers.UpdateOne(MongoCtx, bson.M{"_id": userDB.ID}, change); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.html", gin.H{
			"title": "Settings",
			"error": "An error occured while saving your password",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?setpassword=success")
}

// HandlePOSTAddVolume handles editing (and adding) a volume from POST request
func HandlePOSTEditVolume(c *gin.Context) {
	volumeIdStr := c.PostForm("id")
	volumeName := strings.Trim(c.PostForm("name"), " ")
	volumePath := strings.Trim(c.PostForm("path"), " ")
	volumeIsRecursive := c.PostForm("recursive") == "recursive"
	volumeMediaType := c.PostForm("mediatype") // "Movie" or "TV"

	if volumeIdStr == "" {
		// Adding a volume
		if err := AddVolume(media.Volume{
			ID:          primitive.NewObjectID(),
			Name:        volumeName,
			Path:        volumePath,
			IsRecursive: volumeIsRecursive,
			MediaType:   volumeMediaType,
		}); err != nil {
			RenderHTML(c, http.StatusUnauthorized, "pages/admin_volume.html", gin.H{
				"title":  "Add new volume",
				"volume": media.Volume{},
				"new":    true,
				"error":  err.Error(),
			})
			return
		}
	} else {
		// TODO: editing a volume
		var volume media.Volume
		volumeID, _ := primitive.ObjectIDFromHex(volumeIdStr)
		GetVolumeFromID(volumeID, &volume)
		RenderHTML(c, http.StatusUnauthorized, "pages/admin_volume.html", gin.H{
			"title":  "Edit volume",
			"volume": volume,
			"id":     volume.ID.Hex(),
			"new":    false,
			"error":  "This functionality is not available yet!",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

func HandlePOSTDeleteVolume(c *gin.Context) {
	volumeID := c.PostForm("volumeId")

	err := DeleteVolume(volumeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Volume #%s deleted", volumeID)})
}

func HandlePOSTEditUser(c *gin.Context) {
	userIdStr := c.PostForm("id")
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")
	isAdmin := c.PostForm("isadmin") == "isadmin"

	if userIdStr == "" {
		if err := AddUser(username, password1, password2, isAdmin); err != nil {
			RenderHTML(c, http.StatusUnauthorized, "pages/admin_user.html", gin.H{
				"title":    "Add new user",
				"userEdit": User{},
				"new":      true,
				"error":    err.Error(),
			})
			return
		}
	} else {
		// TODO: editing a user
		var user User
		userID, _ := primitive.ObjectIDFromHex(userIdStr)
		GetUserFromID(userID, &user)
		RenderHTML(c, http.StatusUnauthorized, "pages/admin_user.html", gin.H{
			"title":    "Edit user",
			"userEdit": user,
			"id":       user.ID.Hex(),
			"new":      false,
			"error":    "This functionality is not available yet!",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

func HandlePOSTDeleteUser(c *gin.Context) {
	userID := c.PostForm("userId")

	err := DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("User #%s deleted", userID)})
}
