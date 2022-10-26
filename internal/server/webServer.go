package server

import (
	"encoding/gob"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/pariz/gountries"
	"github.com/samber/lo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
	router.FuncMap = template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},
		"basename":    filepath.Base,
		"countryName": getCountryName,
		"join":        strings.Join,
		"joinStrings": func(sep string, elems ...string) string {
			return strings.Join(lo.Filter(elems, func(elem string, i int) bool {
				return len(elem) > 0
			}), sep)
		},
		"json": func(input any) string {
			ret, _ := json.Marshal(input)
			return string(ret)
		},
		"filmID": func(film media.Film) string {
			return film.ID.Hex()
		},
		"filmName": func(film media.Film) string {
			if film.Title == "" {
				return film.Name
			}
			return film.Title
		},
		"lower": strings.ToLower,
		"personID": func(person media.Person) string {
			return person.ID.Hex()
		},
		"replace":         strings.ReplaceAll,
		"title":           cases.Title(language.English).String,
		"tmdbGetImageURL": tmdb.GetImageURL,
		"getImageURL": func(imageType, key string) string {
			return "/cache/" + imageType + key
		},
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

	// Authentication actions
	mainRouter := router.Use(CheckSetupDone).
		GET("/login", HandleGETLogin).
		POST("/login", HandlePOSTLogin).
		GET("/logout", HandleGETLogout)

	// User needs to be logged in to access these pages
	mainRouter.Use(AuthRequired).
		GET("/", HandleGETIndex).
		GET("/films/*params", HandleGETFilms).
		GET("/film/:id", HandleGETFilm).
		GET("/film/:id/download/:idx", HandleGETDownloadFilm).
		GET("/film/:id/download/:idx/sub/:subIdx", HandleGETDownloadSubtitle).
		GET("/people", HandleGETPeople).
		GET("/person/:id", HandleGetPerson).
		GET("/actor/:id", HandleGetActor).
		GET("/director/:id", HandleGetDirector).
		GET("/writer/:id", HandleGetWriter).
		GET("/settings", HandleGETSettings).
		POST("/setpassword", HandlePOSTSetPassword).
		GET("/cache/*path", HandleGetCache)

	// User needs to be admin to access these pages
	mainRouter.Use(AdminRequired).
		GET("/admin", HandleGETAdmin).
		GET("/admin/volume/:volumeId", HandleGETAdminVolume).
		POST("/admin/editvolume", HandlePOSTEditVolume).
		POST("/admin/deletevolume", HandlePOSTDeleteVolume).
		GET("/admin/user/:userId", HandleGETAdminUser).
		POST("/admin/edituser", HandlePOSTEditUser).
		POST("/admin/deleteuser", HandlePOSTDeleteUser).
		POST("/admin/reloadcache", HandlePOSTReloadCache).
		POST("/admin/editfilmonline", HandlePOSTEditFilmOnline)

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

func getCountryName(code string) string {
	country, _ := gountries.New().FindCountryByAlpha(code)
	return country.Name.Common
}
