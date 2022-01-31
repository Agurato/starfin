package media

import (
	"errors"
	"strconv"

	tmdb "github.com/cyruzin/golang-tmdb"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Movie struct {
	ID          primitive.ObjectID `bson:"_id"`
	Path        string
	Name        string
	ReleaseYear int
	TMDBID      int
	FromVolumes []primitive.ObjectID
	*tmdb.MovieDetails
	*tmdb.MovieCredits
}

// FetchMediaID fetches media ID from TMDB and stores it
func (m *Movie) FetchMediaID() error {
	urlOptions := make(map[string]string)
	if m.ReleaseYear != 0 {
		urlOptions["year"] = strconv.Itoa(m.ReleaseYear)
	}
	tmdbSearchRes, err := TMDBClient.GetSearchMovies(m.Name, urlOptions)
	if err != nil {
		return err
	}
	if len(tmdbSearchRes.Results) == 0 {
		return errors.New("movie not found")
	}
	m.TMDBID = int(tmdbSearchRes.Results[0].ID)
	return nil
}

func (m *Movie) FetchMediaDetails() {
	// Get details
	details, err := TMDBClient.GetMovieDetails(m.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie details from TMDB")
	}
	m.MovieDetails = details

	// Get credits
	credits, err := TMDBClient.GetMovieCredits(m.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie credits from TMDB")
	}
	m.MovieCredits = credits
}

func (m Movie) GetTMDBID() int {
	return m.TMDBID
}
