package media

import log "github.com/sirupsen/logrus"

type Cast struct {
	Character string
	ActorID   int64
}

type Person struct {
	TMDBID   int64
	Name     string
	Photo    string
	Bio      string
	Birthday string
	Deathday string
	IMDbID   string
}

func FetchPersonDetails(personID int64) Person {
	details, err := TMDBClient.GetPersonDetails(int(personID), nil)
	if err != nil {
		log.WithField("personID", personID).Errorln(err)
	}

	return Person{
		TMDBID:   personID,
		Name:     details.Name,
		Photo:    details.ProfilePath,
		Bio:      details.Biography,
		Birthday: details.Birthday,
		Deathday: details.Deathday,
		IMDbID:   details.IMDbID,
	}
}
