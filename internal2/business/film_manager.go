package business

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Agurato/starfin/internal2/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FilmStorer interface {
	GetFilms() []model.Film
	GetFilmsFiltered(years []int, genre, country string) (films []model.Film)

	GetFilmFromID(primitive.ObjectID) (*model.Film, error)
	GetPersonFromTMDBID(int64) (*model.Person, error)
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

type FilmManager interface {
	CacheFilms()

	GetFilms() []model.Film
	GetFilmsFiltered(years []int, genre, country, search string) (films []model.Film)

	GetFilm(filmHexID string) (*model.Film, error)
	GetFilmPath(filmHexID, filmIndex string) (string, error)
	GetFilmSubtitlePath(filmHexID, filmIndex, subtitleIndex string) (string, error)

	EditFilmWithLink(filmID, inputUrl string) error
}

type FilmManagerWrapper struct {
	FilmStorer
	FilmCacher
	FilmDataGetter
	specialChars *regexp.Regexp
}

func NewFilmManagerWrapper(fs FilmStorer, fc FilmCacher, fdg FilmDataGetter) *FilmManagerWrapper {
	return &FilmManagerWrapper{
		FilmStorer:     fs,
		FilmCacher:     fc,
		FilmDataGetter: fdg,
		specialChars:   regexp.MustCompile("[.,\\/#!$%\\^&\\*;:{}=\\-_`~()%\\s\\\\]"),
	}
}

func (fmw FilmManagerWrapper) CacheFilms() {
	films := fmw.FilmStorer.GetFilms()
	for _, film := range films {
		fmw.cachePosterAndBackdrop(&film)
		for _, personID := range film.GetCastAndCrewIDs() {
			person, _ := fmw.GetPersonFromTMDBID(personID)
			fmw.cachePersonPhoto(&person)
		}
	}
}

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

func (fmw FilmManagerWrapper) GetFilms() []model.Film {
	return fmw.FilmStorer.GetFilms()
}

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

func (fmw FilmManagerWrapper) EditFilmWithLink(filmID, inputUrl string) error {
	tmdbID, err := GetTMDBIDFromLink(inputUrl)
	if err != nil {
		return fmt.Errorf("Error getting TMDB ID from URL '%s': %w", inputUrl, err)
	}

	film, err := fmw.GetFilm(filmID)
	if err != nil {
		return fmt.Errorf("Error getting film: %w", err)
	}
	film.TMDBID = int(tmdbID)
	film.FetchDetails()
	err = tryAddFilmToDB(&film, true)
	if err != nil {
		return fmt.Errorf("Could not update film in database: %w", err)
	}

	return nil
}

// cachePosterAndBackdrop caches the poster and the backdrop image of a film
func (fmw FilmManagerWrapper) cachePosterAndBackdrop(film *model.Film) {
	hasToWait, err := fmw.FilmCacher.CachePoster(fmw.FilmDataGetter.GetPosterLink(film.PosterPath), film.PosterPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache poster")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache poster later")
	}
	hasToWait, err = fmw.FilmCacher.CacheBackdrop(fmw.FilmDataGetter.GetPosterLink(film.BackdropPath), film.BackdropPath)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "filmID": film.ID}).Errorln("Could not cache backdrop")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "filmID": film.ID}).Errorln("Will try to cache backdrop later")
	}
}

// cacheCast caches the person's image
func (fmw FilmManagerWrapper) cachePersonPhoto(person *model.Person) {
	hasToWait, err := fmw.FilmCacher.CachePhoto(fmw.FilmDataGetter.GetPhotoLink(person.Photo), person.Photo)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "personTMDBID": person.TMDBID}).Errorln("Could not cache photo")
	}
	if hasToWait {
		log.WithFields(log.Fields{"warning": err, "personTMDBID": person.TMDBID}).Errorln("Will try to cache photo later")
	}
}
