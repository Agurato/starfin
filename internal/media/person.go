package media

import (
	"html/template"
	"strings"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Cast struct {
	Character string
	ActorID   int64
}

type Person struct {
	ID       primitive.ObjectID `bson:"_id"`
	TMDBID   int64
	Name     string
	Photo    string
	Bio      template.HTML
	Birthday string
	Deathday string
	IMDbID   string
}

func FetchPersonDetails(personID int64) Person {
	details, err := TMDBClient.GetPersonDetails(int(personID), nil)
	if err != nil {
		log.WithField("personID", personID).Errorln(err)
		return Person{
			TMDBID: personID,
		}
	}

	return Person{
		TMDBID:   personID,
		Name:     details.Name,
		Photo:    details.ProfilePath,
		Bio:      template.HTML(strings.ReplaceAll(details.Biography, "\n", "<br>")),
		Birthday: details.Birthday,
		Deathday: details.Deathday,
		IMDbID:   details.IMDbID,
	}
}
