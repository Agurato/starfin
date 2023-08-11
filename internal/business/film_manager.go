package business

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Agurato/starfin/internal/model"
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

// FilmFilterer holds the different filters that can be applied
type FilmFilterer interface {
	AddFilm(films *model.Film)
}

type FilmManager struct {
	FilmStorer
	FilmCacher
	FilmMetadataGetter
	FilmFilterer
	specialChars *regexp.Regexp
}

func NewFilmManager(fs FilmStorer, fc FilmCacher, fdg FilmMetadataGetter, ff *Filterer) *FilmManager {
	return &FilmManager{
		FilmStorer:         fs,
		FilmCacher:         fc,
		FilmMetadataGetter: fdg,
		FilmFilterer:       ff,
		specialChars:       regexp.MustCompile("[.,\\/#!$%\\^&\\*;:{}=\\-_~()%\\s\\\\]"),
	}
}

func (fm FilmManager) CacheFilms() {
	films, _ := fm.FilmStorer.GetFilms()
	for _, film := range films {
		fm.cachePosterAndBackdrop(&film)
		for _, personID := range film.GetCastAndCrewIDs() {
			person, _ := fm.GetPersonFromTMDBID(personID)
			fm.cachePersonPhoto(person)
		}
	}
}

// GetFilm returns a Film from its hexadecimal ID
func (fm FilmManager) GetFilm(filmHexID string) (*model.Film, error) {
	filmId, err := primitive.ObjectIDFromHex(filmHexID)
	if err != nil {
		return nil, fmt.Errorf("incorrect film ID: %w", err)
	}
	film, err := fm.FilmStorer.GetFilmFromID(filmId)
	if err != nil {
		return nil, fmt.Errorf("could not get film from ID '%s': %w", filmHexID, err)
	}
	return film, nil
}

// GetFilmPath returns the filepath to a film given its hexadecimal ID and its index in the volume file slice
func (fm FilmManager) GetFilmPath(filmHexID, filmIndex string) (string, error) {
	film, err := fm.GetFilm(filmHexID)
	if err != nil {
		return "", err
	}
	fileIndex, err := strconv.Atoi(filmIndex)
	if err != nil {
		return "", fmt.Errorf("cannot parse film index '%s': %w", filmIndex, err)
	}

	if fileIndex >= len(film.VolumeFiles) {
		fileIndex = len(film.VolumeFiles) - 1
	}
	return film.VolumeFiles[fileIndex].Path, nil
}

// GetFilmSubtitlePath returns the filepath to a subtitle for a film given the film's hexadecimal ID, the film's index in the volume file slice, and the subtitle's index in the external subtitle slice
func (fm FilmManager) GetFilmSubtitlePath(filmHexID, filmIndex, subtitleIndex string) (string, error) {
	film, err := fm.GetFilm(filmHexID)
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
		return "", fmt.Errorf("this film file index does not exist: %d/%d", filmFileIndex, len(film.VolumeFiles))
	}
	extSubtitles := film.VolumeFiles[filmFileIndex].ExtSubtitles
	if subFileIndex >= len(extSubtitles) {
		return "", fmt.Errorf("this film subtitle file index does not exist: %d/%d", subFileIndex, len(extSubtitles))
	}

	return extSubtitles[subFileIndex].Path, nil
}

// GetFilms returns the full slice of films in the database
func (fm FilmManager) GetFilms() []model.Film {
	films, _ := fm.FilmStorer.GetFilms()
	return films
}

// GetFilmsFiltered returns a slice of films, filtered with years of release date, genre, country, and search terms
func (fm FilmManager) GetFilmsFiltered(years []int, genre, country, search string) []model.Film {
	films := fm.FilmStorer.GetFilmsFiltered(years, genre, country)

	search = strings.Trim(search, " ")

	var filteredFilms []model.Film

	search = fm.specialChars.ReplaceAllString(strings.ToLower(search), "")
	for _, m := range films {
		title := fm.specialChars.ReplaceAllString(strings.ToLower(m.Title), "")
		originalTitle := fm.specialChars.ReplaceAllString(strings.ToLower(m.OriginalTitle), "")
		if strings.Contains(title, search) || strings.Contains(originalTitle, search) {
			filteredFilms = append(filteredFilms, m)
		}
	}

	return filteredFilms
}

func (fm FilmManager) GetFilmsWithActor(actorID int64) (films []model.Film) {
	return fm.FilmStorer.GetFilmsWithActor(actorID)
}
func (fm FilmManager) GetFilmsWithDirector(directorID int64) (films []model.Film) {
	return fm.FilmStorer.GetFilmsWithDirector(directorID)
}
func (fm FilmManager) GetFilmsWithWriter(writerID int64) (films []model.Film) {
	return fm.FilmStorer.GetFilmsWithWriter(writerID)
}

func (fm FilmManager) EditFilmWithLink(filmID, inputUrl string) error {
	tmdbID, err := fm.FilmMetadataGetter.GetTMDBIDFromLink(inputUrl)
	if err != nil {
		return fmt.Errorf("error getting TMDB ID from URL '%s': %w", inputUrl, err)
	}

	film, err := fm.GetFilm(filmID)
	if err != nil {
		return fmt.Errorf("error getting film: %w", err)
	}
	film.TMDBID = tmdbID
	fm.FilmMetadataGetter.UpdateFilmDetails(film)
	err = fm.AddFilm(film, true)
	if err != nil {
		return fmt.Errorf("could not update film in database: %w", err)
	}

	return nil
}

func (fm FilmManager) AddFilm(film *model.Film, update bool) error {
	if update || film.TMDBID == 0 || !fm.FilmStorer.IsFilmPresent(film) {
		if err := fm.FilmStorer.AddFilm(film); err != nil {
			return errors.New("cannot add film to database")
		}
		fm.FilmFilterer.AddFilm(film)
		// Cache poster, backdrop
		go fm.cachePosterAndBackdrop(film)
	} else {
		if err := fm.FilmStorer.AddVolumeSourceToFilm(film); err != nil {
			return errors.New("cannot add volume source to film in database")
		}
	}

	for _, personID := range film.GetCastAndCrewIDs() {
		if !fm.FilmStorer.IsPersonPresent(personID) {
			person := fm.FilmMetadataGetter.GetPersonDetails(personID)
			fm.FilmStorer.AddPerson(person)
			// Cache photos
			go fm.cachePersonPhoto(person)
		}
	}

	return nil
}

// cachePosterAndBackdrop caches the poster and the backdrop image of a film
func (fm FilmManager) cachePosterAndBackdrop(film *model.Film) {
	hasToWait, err := fm.FilmCacher.CachePoster(fm.FilmMetadataGetter.GetPosterLink(film.PosterPath), film.PosterPath)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache poster")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache poster later")
	}
	hasToWait, err = fm.FilmCacher.CacheBackdrop(fm.FilmMetadataGetter.GetPosterLink(film.BackdropPath), film.BackdropPath)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache backdrop")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache backdrop later")
	}
}

// cacheCast caches the person's image
func (fm FilmManager) cachePersonPhoto(person *model.Person) {
	hasToWait, err := fm.FilmCacher.CachePhoto(fm.FilmMetadataGetter.GetPhotoLink(person.Photo), person.Photo)
	if err != nil {
		// log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
	}
	if hasToWait {
		// log.WithFields(log.Fields{"warning": err, "personTMDBID": person.TMDBID}).Errorln("Will try to cache photo later")
	}
}
