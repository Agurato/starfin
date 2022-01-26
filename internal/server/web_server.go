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
	// TODO: change to release for deployment
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.SetTrustedProxies(nil)

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

	// Start pages
	router.GET("/start", HandleGETStart)
	router.POST("/start", HandlePOSTStart)

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
		needsLogin.POST("/setpassword", HandlePOSTSetPassword)
	}

	needsAdmin := router.Group("/")
	needsAdmin.Use(AdminRequired)
	{
		needsAdmin.GET("/admin", HandleGETAdmin)
		needsAdmin.GET("/admin/volume/:volumeId", HandleGETVolume)
		needsAdmin.POST("/admin/editvolume", HandlePOSTEditVolume)

		needsAdmin.GET("/adduser")
		needsAdmin.POST("/adduser")
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
			"isAdmin":    false,
			"name":       "",
		}
	} else {
		realUser := user.(User)
		obj["user"] = gin.H{
			"isLoggedIn": true,
			"isAdmin":    realUser.IsAdmin,
			"name":       realUser.Name,
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

// AuthRequired ensures that a request will be aborted if the user is not authenticated
func AdminRequired(c *gin.Context) {
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
	if !user.(User).IsAdmin {
		c.AbortWithStatus(http.StatusUnauthorized)
		RenderHTML(c, http.StatusUnauthorized, "pages/index.html", gin.H{
			"title": "down-low-d",
			"error": "You need to be admin to use this functionality",
		})
		return
	}
	c.Next()
}
