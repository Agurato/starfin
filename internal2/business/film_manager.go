package business

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Agurato/starfin/internal2/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FilmStorer interface {
	GetFilms() ([]model.Film, error)
	GetFilmsFiltered(years []int, genre, country string) (films []model.Film)
	GetFilmsWithActor(actorID int64) (films []model.Film)
	GetFilmsWithDirector(directorID int64) (films []model.Film)
	GetFilmsWithWriter(writerID int64) (films []model.Film)

	GetFilmFromID(primitive.ObjectID) (*model.Film, error)
	GetPersonFromTMDBID(int64) (*model.Person, error)

	IsFilmPresent(film *model.Film) bool
	AddFilm(film *model.Film) error

	AddVolumeSourceToFilm(film *model.Film) error

	IsPersonPresent(personID int64) bool
	AddPerson(person *model.Person)
}

type FilmCacher interface {
	CachePoster(link, key string) (bool, error)
	CacheBackdrop(link, key string) (bool, error)
	CachePhoto(link, key string) (bool, error)
}

type FilmMetadataGetter interface {
	GetPosterLink(key string) string
	GetBackdropLink(key string) string
	GetPhotoLink(key string) string

	GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error)
	GetPersonDetails(personID int64) *model.Person
	UpdateFilmDetails(film *model.Film)
}

type TMDB interface {
}

type FilmManager interface {
	CacheFilms()

	GetFilms() []model.Film
	GetFilmsFiltered(years []int, genre, country, search string) (films []model.Film)
	GetFilmsWithActor(actorID int64) (films []model.Film)
	GetFilmsWithDirector(directorID int64) (films []model.Film)
	GetFilmsWithWriter(writerID int64) (films []model.Film)

	GetFilm(filmHexID string) (*model.Film, error)
	GetFilmPath(filmHexID, filmIndex string) (string, error)
	GetFilmSubtitlePath(filmHexID, filmIndex, subtitleIndex string) (string, error)

	EditFilmWithLink(filmID, inputUrl string) error

	AddFilm(film *model.Film, update bool) error
}

type FilmManagerWrapper struct {
	FilmStorer
	FilmCacher
	FilmMetadataGetter
	Filterer
	TMDB
	specialChars *regexp.Regexp
}

func NewFilmManagerWrapper(fs FilmStorer, fc FilmCacher, fdg FilmMetadataGetter, f Filterer) *FilmManagerWrapper {
	return &FilmManagerWrapper{
		FilmStorer:         fs,
		FilmCacher:         fc,
		FilmMetadataGetter: fdg,
		Filterer:           f,
		specialChars:       regexp.MustCompile("[.,\\/#!$%\\^&\\*;:{}=\\-_`~()%\\s\\\\]"),
	}
}

func (fmw FilmManagerWrapper) CacheFilms() {
	films, _ := fmw.FilmStorer.GetFilms()
	for _, film := range films {
		fmw.cachePosterAndBackdrop(&film)
		for _, personID := range film.GetCastAndCrewIDs() {
			person, _ := fmw.GetPersonFromTMDBID(personID)
			fmw.cachePersonPhoto(person)
		}
	}
}

// GetFilm returns a Film from its hexadecimal ID
func (fmw FilmManagerWrapper) GetFilm(filmHexID string) (*model.Film, error) {
	filmId, err := primitive.ObjectIDFromHex(filmHexID)
	if err != nil {
		return nil, fmt.Errorf("Incorrect film ID: %w", err)
	}
	film, err := fmw.FilmStorer.GetFilmFromID(filmId)
	if err != nil {
		return nil, fmt.Errorf("Could not get film from ID '%s': %w", filmHexID, err)
	}
	return film, nil
}

// GetFilmPath returns the filepath to a film given its hexadecimal ID and its index in the volume file slice
func (fmw FilmManagerWrapper) GetFilmPath(filmHexID, filmIndex string) (string, error) {
	film, err := fmw.GetFilm(filmHexID)
	if err != nil {
		return "", err
	}
	fileIndex, err := strconv.Atoi(filmIndex)
	if err != nil {
		return "", fmt.Errorf("Cannot parse film index '%s': %w", filmIndex, err)
	}

	if fileIndex >= len(film.VolumeFiles) {
		fileIndex = len(film.VolumeFiles) - 1
	}
	return film.VolumeFiles[fileIndex].Path, nil
}

// GetFilmSubtitlePath returns the filepath to a subtitle for a film given the film's hexadecimal ID, the film's index in the volume file slice, and the subtitle's index in the external subtitle slice
func (fmw FilmManagerWrapper) GetFilmSubtitlePath(filmHexID, filmIndex, subtitleIndex string) (string, error) {
	film, err := fmw.GetFilm(filmHexID)
	if err != nil {
		return "", err
	}

	filmFileIndex, err := strconv.Atoi(filmIndex)
	if err != nil {
		filmFileIndex = 0
	}
	subFileIndex, err := strconv.Atoi(subtitleIndex)
	if err != nil {
		subFileIndex = 0
	}
	if filmFileIndex >= len(film.VolumeFiles) {
		return "", fmt.Errorf("This film file index does not exist: %d/%d", filmFileIndex, len(film.VolumeFiles))
	}
	extSubtitles := film.VolumeFiles[filmFileIndex].ExtSubtitles
	if subFileIndex >= len(extSubtitles) {
		return "", fmt.Errorf("This film subtitle file index does not exist: %d/%d", subFileIndex, len(extSubtitles))
	}

	return extSubtitles[subFileIndex].Path, nil
}

// GetFilms returns the full slice of films in the database
func (fmw FilmManagerWrapper) GetFilms() []model.Film {
	films, _ := fmw.FilmStorer.GetFilms()
	return films
}

// GetFilmsFiltered returns a slice of films, filtered with years of release date, genre, country, and search terms
func (fmw FilmManagerWrapper) GetFilmsFiltered(years []int, genre, country, search string) []model.Film {
	films := fmw.FilmStorer.GetFilmsFiltered(years, genre, country)

	search = strings.Trim(search, " ")

	var filteredFilms []model.Film

	search = fmw.specialChars.ReplaceAllString(strings.ToLower(search), "")
	for _, m := range films {
		title := fmw.specialChars.ReplaceAllString(strings.ToLower(m.Title), "")
		originalTitle := fmw.specialChars.ReplaceAllString(strings.ToLower(m.OriginalTitle), "")
		if strings.Contains(title, search) || strings.Contains(originalTitle, search) {
			filteredFilms = append(filteredFilms, m)
		}
	}

	return filteredFilms
}

func (fmw FilmManagerWrapper) GetFilmsWithActor(actorID int64) (films []model.Film) {
	return fmw.FilmStorer.GetFilmsWithActor(actorID)
}
func (fmw FilmManagerWrapper) GetFilmsWithDirector(directorID int64) (films []model.Film) {
	return fmw.FilmStorer.GetFilmsWithDirector(directorID)
}
func (fmw FilmManagerWrapper) GetFilmsWithWriter(writerID int64) (films []model.Film) {
	return fmw.FilmStorer.GetFilmsWithWriter(writerID)
}

func (fmw FilmManagerWrapper) EditFilmWithLink(filmID, inputUrl string) error {
	tmdbID, err := fmw.FilmMetadataGetter.GetTMDBIDFromLink(inputUrl)
	if err != nil {
		return fmt.Errorf("Error getting TMDB ID from URL '%s': %w", inputUrl, err)
	}

	film, err := fmw.GetFilm(filmID)
	if err != nil {
		return fmt.Errorf("Error getting film: %w", err)
	}
	film.TMDBID = int(tmdbID)
	fmw.FilmMetadataGetter.UpdateFilmDetails(film)
	err = fmw.AddFilm(film, true)
	if err != nil {
		return fmt.Errorf("Could not update film in database: %w", err)
	}

	return nil
}

func (fmw FilmManagerWrapper) AddFilm(film *model.Film, update bool) error {
	if update || film.TMDBID == 0 || !fmw.FilmStorer.IsFilmPresent(film) {
		if err := fmw.FilmStorer.AddFilm(film); err != nil {
			return errors.New("cannot add film to database")
		}
		fmw.Filterer.AddFilm(film)
		// Cache poster, backdrop
		go fmw.cachePosterAndBackdrop(film)
	} else {
		if err := fmw.FilmStorer.AddVolumeSourceToFilm(film); err != nil {
			return errors.New("cannot add volume source to film in database")
		}
	}

	for _, personID := range film.GetCastAndCrewIDs() {
		if !fmw.FilmStorer.IsPersonPresent(personID) {
			person := fmw.FilmMetadataGetter.GetPersonDetails(personID)
			fmw.FilmStorer.AddPerson(person)
			// Cache photos
			go fmw.cachePersonPhoto(person)
		}
	}

	return nil
}

// cachePosterAndBackdrop caches the poster and the backdrop image of a film
func (fmw FilmManagerWrapper) cachePosterAndBackdrop(film *model.Film) {
	hasToWait, err := fmw.FilmCacher.CachePoster(fmw.FilmMetadataGetter.GetPosterLink(film.PosterPath), film.PosterPath)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache poster")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache poster later")
	}
	hasToWait, err = fmw.FilmCacher.CacheBackdrop(fmw.FilmMetadataGetter.GetPosterLink(film.BackdropPath), film.BackdropPath)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache backdrop")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache backdrop later")
	}
}

// cacheCast caches the person's image
func (fmw FilmManagerWrapper) cachePersonPhoto(person *model.Person) {
	hasToWait, err := fmw.FilmCacher.CachePhoto(fmw.FilmMetadataGetter.GetPhotoLink(person.Photo), person.Photo)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "personTMDBID": person.TMDBID}).Errorln("Will try to cache photo later")
	}
}
