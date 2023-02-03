package server

import (
	"net/http"
	"strconv"

	"github.com/Agurato/starfin/internal2/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonStorer interface {
	GetPeople() (people []model.Person)
	GetPersonFromID(ID primitive.ObjectID) (person model.Person, err error)

	GetFilmsWithActor(actorID int64) (films []model.Film)
	GetFilmsWithDirector(directorID int64) (films []model.Film)
	GetFilmsWithWriter(writerID int64) (films []model.Film)
}

type PersonHandler struct {
	PersonStorer
}

func NewPersonHandler(ps PersonStorer) *PersonHandler {
	return &PersonHandler{
		PersonStorer: ps,
	}
}

// HandleGETFilms displays the list of people
func (ph PersonHandler) GetPeople(c *gin.Context) {
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

	people := ph.PersonStorer.GetPeople()

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
func (ph PersonHandler) GetPerson(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := ph.PersonStorer.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonStorer.GetFilmsWithActor(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "actor",
		"person": person,
		"films":  films,
	})
}

// HandleGetActor displays the actor's bio and the films they star in
func (ph PersonHandler) GetActor(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := ph.PersonStorer.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonStorer.GetFilmsWithActor(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "actor",
		"person": person,
		"films":  films,
	})
}

// HandleGetDirector displays the directors's bio and the films they directed
func (ph PersonHandler) GetDirector(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := ph.PersonStorer.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonStorer.GetFilmsWithDirector(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "director",
		"person": person,
		"films":  films,
	})
}

// HandleGetWriter displays the writer's bio and the films they wrote
func (ph PersonHandler) GetWriter(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}
	person, err := ph.PersonStorer.GetPersonFromID(id)
	if err != nil {
		RenderHTML(c, http.StatusNotFound, "pages/404.go.html", gin.H{
			"title": "404 - Not Found",
		})
		return
	}

	films := ph.PersonStorer.GetFilmsWithWriter(person.TMDBID)

	RenderHTML(c, http.StatusOK, "pages/person.go.html", gin.H{
		"title":  person.Name,
		"job":    "writer",
		"person": person,
		"films":  films,
	})
}
