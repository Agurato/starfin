package business

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Agurato/starfin/internal/model"
)

type PersonStorer interface {
	GetPeople() ([]model.Person, error)
	GetPersonFromID(ID primitive.ObjectID) (*model.Person, error)
	GetPersonFromTMDBID(ID int64) (*model.Person, error)
}

type PersonManager struct {
	PersonStorer
}

func NewPersonManager(ps PersonStorer) *PersonManager {
	return &PersonManager{
		PersonStorer: ps,
	}
}

func (pm PersonManager) GetPeople() []model.Person {
	people, _ := pm.PersonStorer.GetPeople()
	return people
}

// GetPerson returns a Person from its hexadecimal ID
func (pm PersonManager) GetPerson(personHexID string) (*model.Person, error) {
	personId, err := primitive.ObjectIDFromHex(personHexID)
	if err != nil {
		return nil, fmt.Errorf("incorrect person ID: %w", err)
	}
	person, err := pm.PersonStorer.GetPersonFromID(personId)
	if err != nil {
		return nil, fmt.Errorf("could not get person from ID '%s': %w", personHexID, err)
	}
	return person, nil
}

// GetFilmStaff returns slices of cast, directors, and writers who worked on the film
func (pm PersonManager) GetFilmStaff(film *model.Film) (cast []model.Cast, directors []model.Person, writers []model.Person, err error) {
	for _, character := range film.Characters {
		actor, err := pm.PersonStorer.GetPersonFromTMDBID(character.ActorID)
		if err != nil {
			actor = &model.Person{}
		}
		cast = append(cast, model.Cast{CharacterName: character.CharacterName, Actor: *actor})
	}
	for _, directorID := range film.Directors {
		person, err := pm.PersonStorer.GetPersonFromTMDBID(directorID)
		if err == nil {
			directors = append(directors, *person)
		}
	}
	for _, writerID := range film.Writers {
		person, err := pm.PersonStorer.GetPersonFromTMDBID(writerID)
		if err == nil {
			writers = append(writers, *person)
		}
	}

	return cast, directors, writers, nil
}
