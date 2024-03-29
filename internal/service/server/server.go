package server

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/Agurato/starfin/internal/model"
)

const (
	// UserKey is the session key for user data
	UserKey = "user"
)

var (
	setupDone bool
)

type OwnerStorer interface {
	IsOwnerPresent() (bool, error)
}

// NewServer initializes the server
func NewServer(cookieSecret string, mainHandler *MainHandler, adminHandler *AdminHandler, filmHandler *FilmHandler, personHandler *PersonHandler, rarbgHandler *RarbgHandler, db OwnerStorer) *gin.Engine {
	// Set Gin to production mode
	// TODO: change to release for deployment
	// gin.SetMode(gin.DebugMode)

	router := gin.Default()

	_ = router.SetTrustedProxies(nil)

	// Cookies
	store := cookie.NewStore([]byte(cookieSecret))
	gob.Register(model.User{})
	router.Use(sessions.Sessions("user-session", store))

	// Add template functions
	router.FuncMap = template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},
		"basename":    filepath.Base,
		"countryName": filmHandler.Filterer.GetCountryName,
		"dispDate": func(dt time.Time) string {
			return dt.Format("02 Jan 2006")
		},
		"join": strings.Join,
		"joinStrings": func(sep string, elems ...string) string {
			return strings.Join(lo.Filter(elems, func(elem string, i int) bool {
				return len(elem) > 0
			}), sep)
		},
		"json": func(input any) string {
			ret, _ := json.Marshal(input)
			return string(ret)
		},
		"fileSize": func(size int64) string {
			const unit = 1000
			if size < unit {
				return fmt.Sprintf("%d B", size)
			}
			div, exp := int64(unit), 0
			for n := size / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "kMGTPE"[exp])
		},
		"filmID": func(film model.Film) string {
			return film.ID.Hex()
		},
		"filmName": func(film model.Film) string {
			if film.Title == "" {
				return film.Name
			}
			return film.Title
		},
		"hexID": func(id primitive.ObjectID) string {
			return id.Hex()
		},
		"lower": strings.ToLower,
		"personID": func(person model.Person) string {
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
	router.LoadHTMLGlob("web/templates/**/*.go.html")

	// Static files
	router.Static("/static", "./web/static")
	// 404
	router.NoRoute(mainHandler.Error404)

	// Start page
	router.GET("/start", mainHandler.GETStart)
	router.POST("/start", mainHandler.POSTStart)

	mainRouter := router.Group("/")

	// Authentication actions
	mainRouter.Use(checkSetupDone).
		GET("/login", mainHandler.GETLogin).
		POST("/login", mainHandler.POSTLogin).
		GET("/logout", mainHandler.GETLogout)

	// User needs to be logged in to access these pages
	mainRouter.Use(authRequired).
		GET("/", mainHandler.GETIndex).
		GET("/films/*params", filmHandler.GETFilms).
		GET("/film/:id", filmHandler.GETFilm).
		GET("/film/:id/download/:idx", filmHandler.GETFilmDownload).
		GET("/film/:id/download/:idx/sub/:subIdx", filmHandler.GETSubtitleDownload).
		GET("/people", personHandler.GETPeople).
		GET("/person/:id", personHandler.GETPerson).
		GET("/actor/:id", personHandler.GETActor).
		GET("/director/:id", personHandler.GETDirector).
		GET("/writer/:id", personHandler.GETWriter).
		GET("/settings", mainHandler.GETSettings).
		POST("/setpassword", mainHandler.POSTSetPassword).
		GET("/cache/*path", mainHandler.GETCache)

	if rarbgHandler != nil {
		mainRouter.Use(authRequired).
			GET("/torrents", rarbgHandler.GETTorrents)

		torznab := router.Group("/torznab")
		torznab.GET("/api", rarbgHandler.GETTorznab)
	}

	// User needs to be admin to access these pages
	mainRouter.Use(adminRequired).
		GET("/admin", adminHandler.GETAdmin).
		GET("/admin/volume/:volumeId", adminHandler.GETAdminVolume).
		POST("/admin/editvolume", adminHandler.POSTEditVolume).
		POST("/admin/deletevolume", adminHandler.POSTDeleteVolume).
		GET("/admin/user/:userId", adminHandler.GETAdminUser).
		POST("/admin/edituser", adminHandler.POSTEditUser).
		POST("/admin/deleteuser", adminHandler.POSTDeleteUser).
		POST("/admin/reloadcache", adminHandler.POSTReloadCache).
		POST("/admin/editfilmonline", adminHandler.POSTEditFilmOnline)

	var err error
	setupDone, err = db.IsOwnerPresent()
	if err != nil {
		log.Fatal().Err(err).Msg("error while checking for owner")
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
		realUser := user.(model.User)
		obj["user"] = gin.H{
			"isLoggedIn": true,
			"isAdmin":    realUser.IsAdmin,
			"name":       realUser.Name,
		}
	}
	c.HTML(code, name, obj)
}

// CheckSetupDone ensures that the setup has been done once (a user is registered in the database)
func checkSetupDone(c *gin.Context) {
	if !setupDone {
		c.Redirect(http.StatusSeeOther, "/start")
		c.Abort()
		return
	}
	c.Next()
}

// AuthRequired ensures that a request will be aborted if the user is not authenticated
func authRequired(c *gin.Context) {
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
func adminRequired(c *gin.Context) {
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
	if !user.(model.User).IsAdmin {
		c.AbortWithStatus(http.StatusUnauthorized)
		RenderHTML(c, http.StatusUnauthorized, "pages/index.go.html", gin.H{
			"title": "starfin",
			"error": "You need to be admin to use this functionality",
		})
		return
	}
	c.Next()
}
