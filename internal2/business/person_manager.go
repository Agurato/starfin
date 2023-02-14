package business

import (
	"github.com/Agurato/starfin/internal2/model"
)

type PersonStorer interface {
	GetPersonFromTMDBID(ID int64) (*model.Person, error)
}

type PersonManager interface {
	GetFilmStaff(*model.Film) ([]model.Cast, []model.Person, []model.Person, error)
}

type PersonManagerWrapper struct {
	PersonStorer
}

func NewPersonManagerWrapper(ps PersonStorer) *PersonManagerWrapper {
	return &PersonManagerWrapper{
		PersonStorer: ps,
	}
}

func (pmw PersonManagerWrapper) GetFilmStaff(film *model.Film) (cast []model.Cast, directors []model.Person, writers []model.Person, err error) {
	for _, character := range film.Characters {
		actor, err := pmw.PersonStorer.GetPersonFromTMDBID(character.ActorID)
		if err != nil {
			actor = &model.Person{}
		}
		cast = append(cast, model.Cast{CharacterName: character.CharacterName, Actor: *actor})
	}
	for _, directorID := range film.Directors {
		person, err := pmw.PersonStorer.GetPersonFromTMDBID(directorID)
		if err == nil {
			directors = append(directors, *person)
		}
	}
	for _, writerID := range film.Writers {
		person, err := pmw.PersonStorer.GetPersonFromTMDBID(writerID)
		if err == nil {
			writers = append(writers, *person)
		}
	}

	return nil, nil, nil, nil
}
