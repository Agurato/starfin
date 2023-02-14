package business

import (
	"fmt"

	"github.com/Agurato/starfin/internal2/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PersonStorer interface {
	GetPeople() []model.Person
	GetPersonFromID(ID primitive.ObjectID) (*model.Person, error)
	GetPersonFromTMDBID(ID int64) (*model.Person, error)
}

type PersonManager interface {
	GetPeople() []model.Person

	GetPerson(personHexID string) (*model.Person, error)

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

func (fmw PersonManagerWrapper) GetPeople() []model.Person {
	return fmw.PersonStorer.GetPeople()
}

// GetPerson returns a Person from its hexadecimal ID
func (fmw PersonManagerWrapper) GetPerson(personHexID string) (*model.Person, error) {
	personId, err := primitive.ObjectIDFromHex(personHexID)
	if err != nil {
		return nil, fmt.Errorf("Incorrect person ID: %w", err)
	}
	person, err := fmw.PersonStorer.GetPersonFromID(personId)
	if err != nil {
		return nil, fmt.Errorf("Could not get person from ID '%s': %w", personHexID, err)
	}
	return person, nil
}

// GetFilmStaff returns slices of cast, directors, and writers who worked on the film
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

	return cast, directors, writers, nil
}
