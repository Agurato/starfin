package server

import (
	"net/http"
	"strings"

	"github.com/Agurato/starfin/internal/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type MainCacher interface {
	GetCachedPath(filePath string) string
}

type MainUserManager interface {
	IsOwnerPresent() (bool, error)
	CreateOwner(username, password1, password2 string) (*model.User, error)
	CheckLogin(username, password string) (user *model.User, err error)
	SetUserPassword(username, oldPassword, password1, password2 string) error
}

type MainHandler struct {
	MainCacher
	MainUserManager
}

func NewMainHandler(mc MainCacher, mum MainUserManager) *MainHandler {
	return &MainHandler{
		MainCacher:      mc,
		MainUserManager: mum,
	}
}

// Error404 displays the 404 page
func (mh MainHandler) Error404(c *gin.Context) {
	RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
		"title": "404 - Not Found",
	})
}

// GETIndex displays the index page
func (mh MainHandler) GETIndex(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/index.go.html", gin.H{
		"title": "starfin",
	})
}

// GetStart allows regsitration of first user (admin & owner)
func (mh MainHandler) GETStart(c *gin.Context) {
	if ownerPresent, err := mh.MainUserManager.IsOwnerPresent(); err != nil {
		log.Errorln(err)
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	} else if ownerPresent {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	RenderHTML(c, http.StatusOK, "pages/start.go.html", gin.H{
		"title": "Create admin account",
	})
}

// POSTStart handles registration (only available for first account)
func (mh MainHandler) POSTStart(c *gin.Context) {
	session := sessions.Default(c)
	// Fetch username and passwords from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	user, err := mh.MainUserManager.CreateOwner(username, password1, password2)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: always return a JSON response. Redirection will be handled client-side
	setupDone = true
	// Save cookie
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

// GETLogin displays the registration page
func (mh MainHandler) GETLogin(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/login.go.html", gin.H{
		"title": "Login",
	})
}

// POSTLogin handles login from POST request
func (mh MainHandler) POSTLogin(c *gin.Context) {
	session := sessions.Default(c)
	// Fetch username and password from POST data
	username := strings.Trim(c.PostForm("username"), " ")
	password := strings.Trim(c.PostForm("password"), " ")

	user, err := mh.MainUserManager.CheckLogin(username, password)
	// TODO: always return a JSON response. Redirection will be handled client-side
	if err != nil {
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

// GETLogout logs out the user and redirects to index
func (mh MainHandler) GETLogout(c *gin.Context) {
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

// GETSettings displays the user settings page
func (mh MainHandler) GETSettings(c *gin.Context) {
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

// POSTSetPassword handles changing password from POST request
func (mh MainHandler) POSTSetPassword(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(model.User)
	// Fetch username and password from POST data
	oldPassword := strings.Trim(c.PostForm("old-password"), " ")
	password1 := strings.Trim(c.PostForm("password1"), " ")
	password2 := strings.Trim(c.PostForm("password2"), " ")

	if err := mh.MainUserManager.SetUserPassword(user.Name, oldPassword, password1, password2); err != nil {
		RenderHTML(c, http.StatusUnauthorized, "pages/settings.go.html", gin.H{
			"title": "Settings",
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/settings?setpassword=success")
}

// GETCache serves the cached file
func (mh MainHandler) GETCache(c *gin.Context) {
	cachedFilePath := c.Param("path")
	if cachedFilePath == "" {
		c.AbortWithStatus(404)
		return
	}
	http.ServeFile(c.Writer, c.Request, mh.MainCacher.GetCachedPath(cachedFilePath))
}
