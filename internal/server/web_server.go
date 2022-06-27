package server

import (
	"encoding/gob"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/pariz/gountries"
	"github.com/samber/lo"
)

const (
	// UserKey is the session key for user data
	UserKey = "user"
)

var (
	setupDone = false
)

// InitServer initializes the server
func InitServer() *gin.Engine {

	setupDone = IsOwnerInDatabase()

	// Set Gin to production mode
	// TODO: change to release for deployment
	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Cookies
	store := cookie.NewStore([]byte(os.Getenv(EnvCookieSecret)))
	gob.Register(User{})
	router.Use(sessions.Sessions("user-session", store))

	// Add template functions
	router.FuncMap["add"] = func(a int, b int) int {
		return a + b
	}
	router.FuncMap["basename"] = filepath.Base
	router.FuncMap["countryName"] = func(code string) string {
		country, _ := gountries.New().FindCountryByAlpha(code)
		return country.Name.Common
	}
	router.FuncMap["join"] = strings.Join
	router.FuncMap["joinStrings"] = func(sep string, elems ...string) string {
		return strings.Join(lo.Filter(elems, func(elem string, i int) bool {
			return len(elem) > 0
		}), sep)
	}
	router.FuncMap["lower"] = strings.ToLower
	router.FuncMap["replace"] = strings.ReplaceAll
	router.FuncMap["tmdbGetImageURL"] = tmdb.GetImageURL
	// Load templates
	router.LoadHTMLGlob("web/templates/**/*")

	// Static files
	router.Static("/static", "./web/static")
	// 404
	router.NoRoute(Handle404)

	// Start page
	router.GET("/start", HandleGETStart)
	router.POST("/start", HandlePOSTStart)

	mainRouter := router.Group("/")
	mainRouter.Use(CheckSetupDone)
	{
		// Authentication actions
		mainRouter.GET("/login", HandleGETLogin)
		mainRouter.POST("/login", HandlePOSTLogin)
		mainRouter.GET("/logout", HandleGETLogout)
	}

	// User needs to be logged in to access these pages
	needsLogin := mainRouter.Group("/")
	needsLogin.Use(AuthRequired)
	{
		needsLogin.GET("/", HandleGETIndex)

		needsLogin.GET("/movies", HandleGETMovies)
		needsLogin.GET("/movie/:tmdbId", HandleGETMovie)
		needsLogin.GET("/movie/:tmdbId/download/:idx", HandleGETDownloadMovie)
		needsLogin.GET("/movie/:tmdbId/download/:idx/sub/:subIdx", HandleGETDownloadSubtitle)

		needsLogin.GET("/actor/:tmdbId", HandleGetActor)
		needsLogin.GET("/director/:tmdbId", HandleGetDirector)
		needsLogin.GET("/writer/:tmdbId", HandleGetWriter)

		needsLogin.GET("/settings", HandleGETSettings)
		needsLogin.POST("/setpassword", HandlePOSTSetPassword)
	}

	// User needs to be admin to access these pages
	needsAdmin := mainRouter.Group("/")
	needsAdmin.Use(AdminRequired)
	{
		needsAdmin.GET("/admin", HandleGETAdmin)
		needsAdmin.GET("/admin/volume/:volumeId", HandleGETAdminVolume)
		needsAdmin.POST("/admin/editvolume", HandlePOSTEditVolume)
		needsAdmin.POST("/admin/deletevolume", HandlePOSTDeleteVolume)

		needsAdmin.GET("/admin/user/:userId", HandleGETAdminUser)
		needsAdmin.POST("/admin/edituser", HandlePOSTEditUser)
		needsAdmin.POST("/admin/deleteuser", HandlePOSTDeleteUser)
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

// CheckSetupDone ensures that the setup has been done once (a user is registered in the database)
func CheckSetupDone(c *gin.Context) {
	if !setupDone {
		c.Redirect(http.StatusSeeOther, "/start")
		c.Abort()
		return
	}
	c.Next()
}

// AuthRequired ensures that a request will be aborted if the user is not authenticated
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey)
	// Abort request if user is not in cookies
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
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
		RenderHTML(c, http.StatusUnauthorized, "pages/index.go.html", gin.H{
			"title": "starfin",
			"error": "You need to be logged in to use this functionality",
		})
		return
	}
	if !user.(User).IsAdmin {
		c.AbortWithStatus(http.StatusUnauthorized)
		RenderHTML(c, http.StatusUnauthorized, "pages/index.go.html", gin.H{
			"title": "starfin",
			"error": "You need to be admin to use this functionality",
		})
		return
	}
	c.Next()
}
