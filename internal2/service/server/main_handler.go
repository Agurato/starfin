package server

import (
	"net/http"
	"strings"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal/database"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type MainStorer interface {
	GetUserNb() (int64, error)
}

type MainHandler struct {
	MainStorer
}

func NewMainHandler() *MainHandler {
	return &MainHandler{}
}

// Error404 displays the 404 page
func (mh MainHandler) Error404(c *gin.Context) {
	RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
		"title": "404 - Not Found",
	})
}

// HandleGETIndex displays the index page
func (mh MainHandler) HandleGETIndex(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/index.go.html", gin.H{
		"title": "starfin",
	})
}

// GetStart allows regsitration of first user (admin & owner)
func (mh MainHandler) GetStart(c *gin.Context) {
	if userNb, err := mh.MainStorer.GetUserNb(); err != nil {
		log.Errorln(err)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	} else if userNb != 0 {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	RenderHTML(c, http.StatusOK, "pages/start.go.html", gin.H{
		"title": "Create admin account",
	})
}

// PostStart handles registration (only available for first account)
func (mh MainHandler) PostStart(c *gin.Context) {
	session := sessions.Default(c)
	if userNb, err := mh.MainStorer.GetUserNb(); err != nil {
		log.Errorln(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "An error occured …"})
		return
	} else if userNb != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin user has already been created"})
		return
	}
	// Fetch username and passwords from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	user, err := AddUser(username, password1, password2, true)
	if err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/start.go.html", gin.H{
			"title":    "Start",
			"error":    err.Error(),
			"username": username,
		})
		return
	}

	setupDone = true
	// Save cookie
	user.Password = ""
	session.Set(UserKey, user)
	if err := session.Save(); err != nil {
		RenderHTML(c, http.StatusInternalServerError, "pages/login.go.html", gin.H{
			"title":    "Start",
			"error":    "Server had trouble to log you in",
			"username": username,
		})
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin")
}

// HandleGETLogin displays the registration page
func (mh MainHandler) HandleGETLogin(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/login.go.html", gin.H{
		"title": "Login",
	})
}

// HandlePOSTLogin handles login from POST request
func (mh MainHandler) HandlePOSTLogin(c *gin.Context) {
	session := sessions.Default(c)
	// Fetch username and password from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password := strings.Trim(c.PostForm("password"), " ")

	var (
		user *database.User
		err  error
	)
	if user, err = CheckLogin(username, password); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/login.go.html", gin.H{
			"title":    "Login",
			"error":    err.Error(),
			"username": username,
		})
		return
	}

	// Save cookie
	user.Password = ""
	session.Set(UserKey, user)
	if err := session.Save(); err != nil {
		RenderHTML(c, http.StatusInternalServerError, "pages/login.go.html", gin.H{
			"title":    "Login",
			"error":    "Server had trouble to log you in",
			"username": username,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

// HandleGETLogout logs out the user and redirects to index
func (mh MainHandler) HandleGETLogout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey)

	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The session could not be found"})
		return
	}

	session.Delete(UserKey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server had problem to log you out"})
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// HandleGETSettings displays the user settings page
func (mh MainHandler) HandleGETSettings(c *gin.Context) {
	success := ""
	setPassword := c.Query("setpassword")
	if setPassword == "success" {
		success = "Password changed successfully"
	}
	RenderHTML(c, http.StatusOK, "pages/settings.go.html", gin.H{
		"title":   "Settings",
		"success": success,
	})
}

// HandlePOSTSetPassword handles changing password from POST request
func (mh MainHandler) HandlePOSTSetPassword(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(database.User)
	// Fetch username and password from POST data
	oldPassword := strings.Trim(c.PostForm("old-password"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	if err := SetUserPassword(user.Name, oldPassword, password1, password2); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.go.html", gin.H{
			"title": "Settings",
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?setpassword=success")
}

// HandleGetCache serves the cached file
func (mh MainHandler) HandleGetCache(c *gin.Context) {
	cachedFilePath := c.Param("path")
	if cachedFilePath == "" {
		c.AbortWithStatus(404)
		return
	}
	http.ServeFile(c.Writer, c.Request, cache.GetCachedPath(cachedFilePath))
}
