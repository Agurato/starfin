package server

import (
	"encoding/gob"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

const (
	// UserKey is the session key for user data
	UserKey = "user"
)

// InitServer initializes the server
func InitServer() *gin.Engine {
	// Set Gin to production mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Cookies
	store := cookie.NewStore([]byte(os.Getenv(EnvCookieSecret)))
	gob.Register(User{})
	router.Use(sessions.Sessions("user-session", store))

	// Load templates
	router.LoadHTMLGlob("web/templates/**/*")

	// Static files
	router.Static("/static", "./web/static")
	// 404
	router.NoRoute(Handle404)

	// Basic pages
	router.GET("/", HandleGETIndex)

	// Authentication actions
	// Removed Register handling for production
	// Adding new user is now done via command line (-gen-user)
	// router.GET("/register", HandleGETRegister)
	// router.POST("/register", HandlePOSTRegister)
	router.GET("/login", HandleGETLogin)
	router.POST("/login", HandlePOSTLogin)
	router.GET("/logout", HandleGETLogout)

	// User needs to be logged in to access these pages
	needsLogin := router.Group("/")
	needsLogin.Use(AuthRequired)
	{
		needsLogin.GET("/search", HandleGETSearch)
		needsLogin.GET("/movie/:tmdbId", HandleGETMovie)

		needsLogin.GET("/u/:userId", HandleGETUser)

		needsLogin.GET("/settings", HandleGETSettings)
		router.POST("/setpassword", HandlePOSTSetPassword)
	}

	return router
}

// RenderHTML renders HTML pages and adds useful objects for templates
func RenderHTML(c *gin.Context, code int, name string, obj gin.H) {
	session := sessions.Default(c)
	user := session.Get(UserKey)
	if user == nil {
		obj["user"] = gin.H{
			"isLoggedIn": false,
			"name":       "",
		}
	} else {
		obj["user"] = gin.H{
			"isLoggedIn": true,
			"name":       user.(User).Name,
		}
	}
	c.HTML(code, name, obj)
}

// AuthRequired ensures that a request will be aborted if the user is not authenticated
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey)
	// Abort request if user is not in cookies
	if user == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		RenderHTML(c, http.StatusUnauthorized, "pages/index.html", gin.H{
			"title": "down-low-d",
			"error": "You need to be logged in to use this functionality",
		})
		return
	}
	c.Next()
}
