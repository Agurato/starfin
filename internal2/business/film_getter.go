package business

import (
	"github.com/Agurato/starfin/internal2/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FilmStorer interface {
	GetFilmFromID(primitive.ObjectID) (model.Film, error)
	GetPersonFromTMDBID(int64) (model.Person, error)
	GetFilmsFiltered(years []int, genre, country string) (films []model.Film)
}

type FilmGetter struct {
	FilmStorer
}
