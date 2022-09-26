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

	"github.com/Agurato/starfin/internal/context"
	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
)

const (
	// UserKey is the session key for user data
	UserKey = "user"
)

var (
	setupDone = false

	db database.DB
)

// InitServer initializes the server
func InitServer(datab database.DB) *gin.Engine {
	db = datab
	setupDone = db.IsOwnerPresent()

	// Set Gin to production mode
	// TODO: change to release for deployment
	// gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Cookies
	store := cookie.NewStore([]byte(os.Getenv(context.EnvCookieSecret)))
	gob.Register(database.User{})
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
	router.FuncMap["filmID"] = func(film media.Film) string {
		return film.ID.Hex()
	}
	router.FuncMap["filmName"] = func(film media.Film) string {
		if film.Title == "" {
			return film.Name
		}
		return film.Title
	}
	router.FuncMap["lower"] = strings.ToLower
	router.FuncMap["replace"] = strings.ReplaceAll
	router.FuncMap["tmdbGetImageURL"] = tmdb.GetImageURL
	router.FuncMap["getImageURL"] = func(imageType, key string) string {
		return "/cache/" + imageType + key
	}

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

		needsLogin.GET("/films", HandleGETFilms)
		needsLogin.GET("/films/page/:page", HandleGETFilms)
		needsLogin.GET("/film/:id", HandleGETFilm)
		needsLogin.GET("/film/:id/download/:idx", HandleGETDownloadFilm)
		needsLogin.GET("/film/:id/download/:idx/sub/:subIdx", HandleGETDownloadSubtitle)

		needsLogin.GET("/people", HandleGETPeople)
		needsLogin.GET("/actor/:tmdbId", HandleGetActor)
		needsLogin.GET("/director/:tmdbId", HandleGetDirector)
		needsLogin.GET("/writer/:tmdbId", HandleGetWriter)

		needsLogin.GET("/settings", HandleGETSettings)
		needsLogin.POST("/setpassword", HandlePOSTSetPassword)

		needsLogin.GET("/cache/*path", HandleGetCache)
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
		realUser := user.(database.User)
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
	if !user.(database.User).IsAdmin {
		c.AbortWithStatus(http.StatusUnauthorized)
		RenderHTML(c, http.StatusUnauthorized, "pages/index.go.html", gin.H{
			"title": "starfin",
			"error": "You need to be admin to use this functionality",
		})
		return
	}
	c.Next()
}
