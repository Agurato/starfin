package business

import (
	"fmt"

	"github.com/Agurato/starfin/internal2/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FilmStorer interface {
	GetFilms() []model.Film

	GetFilmFromID(primitive.ObjectID) (*model.Film, error)
	GetPersonFromTMDBID(int64) (model.Person, error)
	GetFilmsFiltered(years []int, genre, country string) (films []model.Film)
}

type FilmCacher interface {
	CachePoster(link, key string) (bool, error)
	CacheBackdrop(link, key string) (bool, error)
	CachePhoto(link, key string) (bool, error)
}

type FilmDataGetter interface {
	GetPosterLink(key string) string
	GetBackdropLink(key string) string
	GetPhotoLink(key string) string
}

type FilmManager struct {
	FilmStorer
	FilmCacher
	FilmDataGetter
}

func NewFilmManager(fs FilmStorer, fc FilmCacher, fdg FilmDataGetter) *FilmManager {
	return &FilmManager{
		FilmStorer: fs,
		FilmCacher: fc,
	}
}

func (fm FilmManager) CacheFilms() {
	films := fm.FilmStorer.GetFilms()
	for _, film := range films {
		fm.cachePosterAndBackdrop(&film)
		for _, personID := range film.GetCastAndCrewIDs() {
			person, _ := fm.GetPersonFromTMDBID(personID)
			fm.cachePersonPhoto(&person)
		}
	}
}

func (fm FilmManager) GetFilm(filmHexID string) (*model.Film, error) {
	filmId, err := primitive.ObjectIDFromHex(filmHexID)
	if err != nil {
		return nil, fmt.Errorf("Incorrect film ID: %w", err)
	}
	film, err := fm.FilmStorer.GetFilmFromID(filmId)
	if err != nil {
		return nil, fmt.Errorf("Could not get film from ID '%s': %w", filmHexID, err)
	}
	return film, nil
}

func (fm FilmManager) EditFilmWithLink(filmID, inputUrl string) error {
	tmdbID, err := GetTMDBIDFromLink(inputUrl)
	if err != nil {
		return fmt.Errorf("Error getting TMDB ID from URL '%s': %w", inputUrl, err)
	}

	film, err := fm.GetFilm(filmID)
	if err != nil {
		return fmt.Errorf("Error getting film: %w", err)
	}
	film.TMDBID = int(tmdbID)
	film.FetchDetails()
	err = tryAddFilmToDB(&film, true)
	if err != nil {
		return fmt.Errorf("Could not update film in database: %w", err)
	}
}

// cachePosterAndBackdrop caches the poster and the backdrop image of a film
func (fm FilmManager) cachePosterAndBackdrop(film *model.Film) {
	hasToWait, err := fm.FilmCacher.CachePoster(fm.FilmDataGetter.GetPosterLink(film.PosterPath), film.PosterPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache poster")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache poster later")
	}
	hasToWait, err = fm.FilmCacher.CacheBackdrop(fm.FilmDataGetter.GetPosterLink(film.BackdropPath), film.BackdropPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache backdrop")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache backdrop later")
	}
}

// cacheCast caches the person's image
func (fm FilmManager) cachePersonPhoto(person *model.Person) {
	hasToWait, err := fm.FilmCacher.CachePhoto(fm.FilmDataGetter.GetPhotoLink(person.Photo), person.Photo)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "personTMDBID": person.TMDBID}).Errorln("Will try to cache photo later")
	}
}
