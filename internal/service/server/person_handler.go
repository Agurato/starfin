package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Agurato/starfin/internal/model"
)

type PersonManager interface {
	GetPeople() []model.Person

	GetPerson(personHexID string) (*model.Person, error)
}

type PersonFilmManager interface {
	GetFilmsWithActor(actorID int64) (films []model.Film)
	GetFilmsWithDirector(directorID int64) (films []model.Film)
	GetFilmsWithWriter(writerID int64) (films []model.Film)
}

type PersonPaginater[T model.Person] interface {
	GetPagination(currentPage int64, items []T) ([]T, []model.Pagination)
}

type PersonHandler struct {
	PersonManager
	PersonFilmManager
	PersonPaginater[model.Person]
}

func NewPersonHandler(pm PersonManager, pfm PersonFilmManager, pp PersonPaginater[model.Person]) *PersonHandler {
	return &PersonHandler{
		PersonManager:     pm,
		PersonFilmManager: pfm,
		PersonPaginater:   pp,
	}
}

// HandleGETFilms displays the list of people
func (ph PersonHandler) GETPeople(c *gin.Context) {
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

	people := ph.PersonManager.GetPeople()

	// Filter films from search
	// if inputSearch, ok = c.GetQuery("search"); ok {
	// 	films, searchTerm, searchYear = SearchFilms(inputSearch, people)
	// }

	people, pages := ph.PersonPaginater.GetPagination(int64(page), people)

	RenderHTML(c, http.StatusOK, "pages/people.go.html", gin.H{
		"title":      "People",
		"people":     people,
		"search":     inputSearch,
		"searchTerm": searchTerm,
		"pages":      pages,
	})
}

// GETPerson displays the actor's bio and the films they star in
func (ph PersonHandler) GETPerson(c *gin.Context) {
	ph.GETActor(c)
}

// GETActor displays the actor's bio and the films they star in
func (ph PersonHandler) GETActor(c *gin.Context) {
	person, err := ph.PersonManager.GetPerson(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonFilmManager.GetFilmsWithActor(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "actor",
		"person": person,
		"films":  films,
	})
}

// HandleGetDirector displays the directors's bio and the films they directed
func (ph PersonHandler) GETDirector(c *gin.Context) {
	person, err := ph.PersonManager.GetPerson(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonFilmManager.GetFilmsWithDirector(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "director",
		"person": person,
		"films":  films,
	})
}

// HandleGetWriter displays the writer's bio and the films they wrote
func (ph PersonHandler) GETWriter(c *gin.Context) {
	person, err := ph.PersonManager.GetPerson(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonFilmManager.GetFilmsWithWriter(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "writer",
		"person": person,
		"films":  films,
	})
}
