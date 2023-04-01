package server

import (
	"fmt"
	"net/http"

	"github.com/Agurato/starfin/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/pariz/gountries"
)

type FilmManager interface {
	GetFilm(filmHexID string) (*model.Film, error)
	GetFilmPath(filmHexID, filmIndex string) (string, error)
	GetFilmSubtitlePath(filmHexID, filmIndex, subtitleIndex string) (string, error)

	GetFilms() []model.Film
	GetFilmsFiltered(years []int, genre, country, search string) (films []model.Film)
}

type FilmPersonManager interface {
	GetFilmStaff(*model.Film) ([]model.Cast, []model.Person, []model.Person, error)
}

type Filterer interface {
	ParseParamsFilters(params string) (yearFilter string, years []int, genre, country string, page int, err error)
	GetCountryName(code string) string

	GetCountries() []string
	GetDecades() []model.Decade
	GetGenres() []string
}

type countryMapping struct {
	Value string
	Code  string
}

type FilmHandler struct {
	FilmManager
	FilmPersonManager
	countries []countryMapping
	Filterer
}

func NewFilmHandler(fm FilmManager, fpm FilmPersonManager, f Filterer) *FilmHandler {
	var countries []countryMapping
	for code, country := range gountries.New().Countries {
		countries = append(countries, countryMapping{
			Value: country.Name.Common,
			Code:  code,
		})
	}
	return &FilmHandler{
		FilmManager:       fm,
		FilmPersonManager: fpm,
		countries:         countries,
		Filterer:          f,
	}
}

// GETFilm displays information about a film
func (fh FilmHandler) GETFilm(c *gin.Context) {
	film, err := fh.FilmManager.GetFilm(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	cast, directors, writers, err := fh.FilmPersonManager.GetFilmStaff(film)

	RenderHTML(c, http.StatusOK, "pages/film.go.html", gin.H{
		"title":     fmt.Sprintf("%s (%d)", film.Title, film.ReleaseYear),
		"film":      film,
		"directors": directors,
		"writers":   writers,
		"cast":      cast,
		"admin": gin.H{
			"genres":    fh.Filterer.GetGenres(),
			"countries": fh.countries,
		},
	})
}

// GETFilmDownload downloads a film file
func (fh FilmHandler) GETFilmDownload(c *gin.Context) {
	filmPath, err := fh.FilmManager.GetFilmPath(c.Param("id"), c.Param("idx"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	http.ServeFile(c.Writer, c.Request, filmPath)
}

// GETSubtitleDownload downloads a subtitle file
func (fh FilmHandler) GETSubtitleDownload(c *gin.Context) {
	subPath, err := fh.FilmManager.GetFilmSubtitlePath(c.Param("id"), c.Param("idx"), c.Param("subIdx"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	http.ServeFile(c.Writer, c.Request, subPath)
}

// GETFilms displays the list of films
func (fh FilmHandler) GETFilms(c *gin.Context) {
	yearFilter, years, genre, country, page, err := fh.Filterer.ParseParamsFilters(c.Param("params"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
	}

	search, _ := c.GetQuery("search")
	films := fh.FilmManager.GetFilmsFiltered(years, genre, country, search)

	films, pages := getPagination(int64(page), films)

	RenderHTML(c, http.StatusOK, "pages/films.go.html", gin.H{
		"title":             "Films",
		"films":             films,
		"filtererCountries": fh.Filterer.GetCountries(),
		"filtererDecades":   fh.Filterer.GetDecades(),
		"filtererGenres":    fh.Filterer.GetGenres(),
		"filterYear":        yearFilter,
		"filterGenre":       genre,
		"filterCountry":     country,
		"search":            search,
		"pages":             pages,
	})
}
