package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/gin-gonic/gin"
	"github.com/pariz/gountries"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FilmStorer interface {
	GetFilmFromID(primitive.ObjectID) (model.Film, error)
	GetPersonFromTMDBID(int64) (model.Person, error)
	GetFilmsFiltered(years []int, genre, country string) (films []model.Film)
}

type FilmGetter interface {
	GetFilm(id string) model.Film
}

type FilmHandler struct {
	FilmStorer
}

func NewFilmHandler(fs FilmStorer) *FilmHandler {
	return &FilmHandler{
		FilmStorer: fs,
	}
}

// GETFilm displays information about a film
func (fh FilmHandler) GETFilm(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	film, err := fh.FilmStorer.GetFilmFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	var (
		fullCast  []gin.H
		directors []model.Person
		writers   []model.Person
	)
	for _, cast := range film.Cast {
		actor, err := fh.FilmStorer.GetPersonFromTMDBID(cast.ActorID)
		if err != nil {
			log.WithField("actorID", cast.ActorID).Errorln("Could not find actor")
			actor = model.Person{}
		}
		fullCast = append(fullCast, gin.H{
			"Character": cast.Character,
			"ID":        actor.ID.Hex(),
			"Name":      actor.Name,
			"Photo":     actor.Photo,
		})
	}
	for _, directorID := range film.Directors {
		person, err := fh.FilmStorer.GetPersonFromTMDBID(directorID)
		if err == nil {
			directors = append(directors, person)
		}
	}
	for _, writerID := range film.Writers {
		person, err := fh.FilmStorer.GetPersonFromTMDBID(writerID)
		if err == nil {
			writers = append(writers, person)
		}
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
		"admin": gin.H{
			"genres":    filters.Genres,
			"countries": countries,
		},
	})
}

// GETFilmDownload downloads a film file
func (fh FilmHandler) GETFilmDownload(c *gin.Context) {
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

	film, err := fh.FilmStorer.GetFilmFromID(id)
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

// GETSubtitleDownload downloads a subtitle file
func (fh FilmHandler) GETSubtitleDownload(c *gin.Context) {
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

	film, err := fh.FilmStorer.GetFilmFromID(id)
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

// GETFilms displays the list of films
func (fh FilmHandler) GETFilms(c *gin.Context) {
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

	films := fh.FilmStorer.GetFilmsFiltered(years, genre, country)

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
