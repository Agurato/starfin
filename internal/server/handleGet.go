package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pariz/gountries"
	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	nbFilmsPerPage int64 = 20
)

// Handle404 displays the 404 page
func Handle404(c *gin.Context) {
	RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
		"title": "404 - Not Found",
	})
}

// HandleGETStart allows regsitration of first user (admin & owner)
func HandleGETStart(c *gin.Context) {
	if userNb, err := db.GetUserNb(); err != nil {
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

// HandleGETIndex displays the index page
func HandleGETIndex(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/index.go.html", gin.H{
		"title": "starfin",
	})
}

// HandleGETLogin displays the registration page
func HandleGETLogin(c *gin.Context) {
	RenderHTML(c, http.StatusOK, "pages/login.go.html", gin.H{
		"title": "Login",
	})
}

// HandleGETLogout logs out the user and redirects to index
func HandleGETLogout(c *gin.Context) {
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

// HandleGETFilm displays information about a film
func HandleGETFilm(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	film, err := db.GetFilmFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	var (
		fullCast  []gin.H
		directors []media.Person
		writers   []media.Person
	)
	for _, cast := range film.Cast {
		actor, err := db.GetPersonFromTMDBID(cast.ActorID)
		if err != nil {
			log.WithField("actorID", cast.ActorID).Errorln("Could not find actor")
			actor = media.Person{}
		}
		fullCast = append(fullCast, gin.H{
			"Character": cast.Character,
			"ID":        actor.ID.Hex(),
			"Name":      actor.Name,
			"Photo":     actor.Photo,
		})
	}
	for _, directorID := range film.Directors {
		person, err := db.GetPersonFromTMDBID(directorID)
		if err == nil {
			directors = append(directors, person)
		}
	}
	for _, writerID := range film.Writers {
		person, err := db.GetPersonFromTMDBID(writerID)
		if err == nil {
			writers = append(writers, person)
		}
	}

	var volumes []string
	for _, path := range film.VolumeFiles {
		var volume media.Volume
		db.GetVolumeFromID(path.FromVolume, &volume)
		volumes = append(volumes, volume.Name)
	}

	var countries []gin.H
	for code, country := range gountries.New().Countries {
		countries = append(countries, gin.H{
			"value": country.Name.Common,
			"code":  code,
		})
	}

	RenderHTML(c, http.StatusOK, "pages/film.go.html", gin.H{
		"title":     fmt.Sprintf("%s (%d)", film.Title, film.ReleaseYear),
		"film":      film,
		"directors": directors,
		"writers":   writers,
		"cast":      fullCast,
		"volumes":   volumes,
		"admin": gin.H{
			"genres":    filters.Genres,
			"countries": countries,
		},
	})
}

// HandleGETDownloadFilm downloads a film file
func HandleGETDownloadFilm(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	fileIndex, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		fileIndex = 0
	}

	film, err := db.GetFilmFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	if fileIndex >= len(film.VolumeFiles) {
		fileIndex = len(film.VolumeFiles) - 1
	}
	http.ServeFile(c.Writer, c.Request, film.VolumeFiles[fileIndex].Path)
}

// HandleGETDownloadSubtitle downloads a subtitle file
func HandleGETDownloadSubtitle(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	filmFileIndex, err := strconv.Atoi(c.Param("idx"))
	if err != nil {
		filmFileIndex = 0
	}
	subFileIndex, err := strconv.Atoi(c.Param("subIdx"))
	if err != nil {
		subFileIndex = 0
	}

	film, err := db.GetFilmFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	if filmFileIndex >= len(film.VolumeFiles) {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	extSubtitles := film.VolumeFiles[filmFileIndex].ExtSubtitles
	if subFileIndex >= len(extSubtitles) {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	http.ServeFile(c.Writer, c.Request, extSubtitles[subFileIndex].Path)
}

// HandleGETFilms displays the list of people
func HandleGETPeople(c *gin.Context) {
	var (
		inputSearch string
		searchTerm  string
		page        int
		err         error
		// ok          bool
	)

	// Get page number
	pageParam := c.Param("page")
	if pageParam == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
				"title": "404 - Not Found",
			})
		}
	}

	people := db.GetPeople()

	// Filter films from search
	// if inputSearch, ok = c.GetQuery("search"); ok {
	// 	films, searchTerm, searchYear = SearchFilms(inputSearch, people)
	// }

	people, pages := getPagination(int64(page), people)

	RenderHTML(c, http.StatusOK, "pages/people.go.html", gin.H{
		"title":      "People",
		"people":     people,
		"search":     inputSearch,
		"searchTerm": searchTerm,
		"pages":      pages,
	})
}

// HandleGetActor displays the actor's bio and the films they star in
func HandleGetPerson(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := db.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := db.GetFilmsWithActor(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "actor",
		"person": person,
		"films":  films,
	})
}

// HandleGetActor displays the actor's bio and the films they star in
func HandleGetActor(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := db.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := db.GetFilmsWithActor(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "actor",
		"person": person,
		"films":  films,
	})
}

// HandleGetDirector displays the directors's bio and the films they directed
func HandleGetDirector(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := db.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := db.GetFilmsWithDirector(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "director",
		"person": person,
		"films":  films,
	})
}

// HandleGetWriter displays the writer's bio and the films they wrote
func HandleGetWriter(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := db.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := db.GetFilmsWithWriter(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "writer",
		"person": person,
		"films":  films,
	})
}

// HandleGETFilms displays the list of films
func HandleGETFilms(c *gin.Context) {
	var (
		inputSearch string
		ok          bool
	)

	yearFilter, years, genre, country, page, err := ParseParamsFilters(c.Param("params"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
	}

	films := db.GetFilmsFiltered(years, genre, country)

	// Filter films from search
	if inputSearch, ok = c.GetQuery("search"); ok {
		films = SearchFilms(inputSearch, films)
	}

	films, pages := getPagination(int64(page), films)

	RenderHTML(c, http.StatusOK, "pages/films.go.html", gin.H{
		"title":         "Films",
		"films":         films,
		"filters":       filters,
		"filterYear":    yearFilter,
		"filterGenre":   genre,
		"filterCountry": country,
		"search":        inputSearch,
		"pages":         pages,
	})
}

// HandleGETSettings displays the user settings page
func HandleGETSettings(c *gin.Context) {
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

// HandleGETUser displays the page of a user
func HandleGETUser(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(UserKey).(database.User)

	var userDB database.User

	// Check for user existence and fetch all queries
	if err := db.GetUserFromID(user.ID, &userDB); err != nil {
		RenderHTML(c, http.StatusOK, "pages/user.go.html", gin.H{
			"title": fmt.Sprintf("%s's profile", c.Param("userId")),
			"error": "User does not exist!",
		})
		return
	}

	RenderHTML(c, http.StatusOK, "pages/user.go.html", gin.H{
		"title": fmt.Sprintf("%s's profile", c.Param("userId")),
		"user":  c.Param("userId"),
	})
}

// HandleGETAdmin displays the admin page
func HandleGETAdmin(c *gin.Context) {
	var (
		volumesWithStringID []gin.H
		usersWithStringID   []gin.H
	)

	volumes, err := db.GetVolumes()
	if err != nil {
		log.Errorln(err)
		RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
			"title":   "Admin",
			"volumes": volumesWithStringID,
			"users":   usersWithStringID,
			"error":   "An error occured …",
		})
	}
	for _, vol := range volumes {
		volumesWithStringID = append(volumesWithStringID, gin.H{
			"id":  vol.ID.Hex(),
			"obj": vol,
		})
	}

	users, err := db.GetUsers()
	if err != nil {
		log.Errorln(err)
		RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
			"title":   "Admin",
			"volumes": volumesWithStringID,
			"users":   usersWithStringID,
			"error":   "An error occured …",
		})
	}
	for _, user := range users {
		usersWithStringID = append(usersWithStringID, gin.H{
			"id":  user.ID.Hex(),
			"obj": user,
		})
	}
	RenderHTML(c, http.StatusOK, "pages/admin.go.html", gin.H{
		"title":   "Admin",
		"volumes": volumesWithStringID,
		"users":   usersWithStringID,
	})
}

// HandleGETAdminVolume displays the volume edit page
func HandleGETAdminVolume(c *gin.Context) {
	volumeIdStr := c.Param("volumeId")

	// If we're adding a new volume
	if volumeIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title":  "Add new volume",
			"volume": media.Volume{},
			"new":    true,
		})
		return
	}

	volumeId, err := primitive.ObjectIDFromHex(volumeIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title": "Edit volume",
			"error": "Incorrect volume ID!",
		})
	}
	var volume media.Volume
	if err := db.GetVolumeFromID(volumeId, &volume); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
			"title": "Edit volume",
			"error": "Volume does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_volume.go.html", gin.H{
		"title":  "Edit volume",
		"volume": volume,
		"id":     volume.ID.Hex(),
		"new":    false,
	})
}

// HandleGETAdminUser displays the user edit page
func HandleGETAdminUser(c *gin.Context) {
	userIdStr := c.Param("userId")

	// If we're adding a new user
	if userIdStr == "new" {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Add new user",
			"user":  database.User{},
			"new":   true,
		})
		return
	}

	userId, err := primitive.ObjectIDFromHex(userIdStr)
	if err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Edit user",
			"error": "Incorrect user ID!",
		})
	}
	var user database.User
	if err := db.GetUserFromID(userId, &user); err != nil {
		RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
			"title": "Edit user",
			"error": "User does not exist!",
		})
		return
	}
	RenderHTML(c, http.StatusOK, "pages/admin_user.go.html", gin.H{
		"title":    "Edit user",
		"userEdit": user,
		"id":       user.ID.Hex(),
		"new":      false,
	})
}

// HandleGetCache serves the cached file
func HandleGetCache(c *gin.Context) {
	cachedFilePath := c.Param("path")
	if cachedFilePath == "" {
		c.AbortWithStatus(404)
		return
	}
	http.ServeFile(c.Writer, c.Request, cache.GetCachedPath(cachedFilePath))
}
