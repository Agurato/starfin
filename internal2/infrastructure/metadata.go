package infrastructure

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal2/model"
	"github.com/PuerkitoBio/goquery"
	tmdb "github.com/cyruzin/golang-tmdb"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

var (
	// TMDBClient holds the client for tmdb access
	TMDBClient *tmdb.Client
)

type Metadata interface {
	GetPosterLink(key string) string
	GetBackdropLink(key string) string
	GetPhotoLink(key string) string
	CachePoster(key string) (bool, error)
	CacheBackdrop(key string) (bool, error)
	CachePhoto(key string) (bool, error)

	GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error)
	GetPersonDetails(personID int64) *model.Person
	UpdateFilmDetails(film *model.Film)
}

type MetadataWrapper struct {
	client *tmdb.Client
}

// NewMetadataWrapper initializes a MetadataWrapper
func NewMetadataWrapper(tmdbAPIKey string) (*MetadataWrapper, error) {
	client, err := tmdb.Init(tmdbAPIKey)
	if err != nil {
		return nil, err
	}
	return &MetadataWrapper{
		client: client,
	}, nil
}

const (
	poster       = "poster"
	backdrop     = "backdrop"
	photo        = "photo"
	tmdbImageURL = "https://image.tmdb.org/t/p/"
)

// GetPosterLink caches a poster from TMDB using its id
func (mw MetadataWrapper) GetPosterLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

// GetBackdropLink caches a backdrop from TMDB using its id
func (mw MetadataWrapper) GetBackdropLink(key string) string {
	return tmdbImageURL + tmdb.W1280 + key
}

// GetPhotoLink caches a person's photo from TMDB using its id
func (mw MetadataWrapper) GetPhotoLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

// CachePoster caches a poster from TMDB using its id
func (mw MetadataWrapper) CachePoster(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, poster+key)
	}
	return false, nil
}

// CacheBackdrop caches a backdrop from TMDB using its id
func (mw MetadataWrapper) CacheBackdrop(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W1280+key, backdrop+key)
	}
	return false, nil
}

// CachePhoto caches a person's photo from TMDB using its id
func (mw MetadataWrapper) CachePhoto(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, photo+key)
	}
	return false, nil
}

func (mw MetadataWrapper) UpdateFilmDetails(film *model.Film) {
	// Get details
	details, err := TMDBClient.GetMovieDetails(film.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": film.TMDBID, "error": err}).Errorln("Unable to fetch film details from TMDB")
	}
	film.IMDbID = details.IMDbID
	film.Title = details.Title
	film.OriginalTitle = details.OriginalTitle
	film.Year = details.ReleaseDate[:4]
	film.Runtime = strconv.Itoa(details.Runtime)
	film.Tagline = details.Tagline
	film.Overview = details.Overview
	film.PosterPath = details.PosterPath
	film.BackdropPath = details.BackdropPath
	film.IMDbRating = mw.getIMDbRating(film.IMDbID)
	film.LetterboxdRating = mw.getLetterboxdRating(film.IMDbID)

	// Set genres
	film.Genres = nil
	for _, genre := range details.Genres {
		film.Genres = append(film.Genres, genre.Name)
	}

	// Set classification
	releaseDates, err := TMDBClient.GetMovieReleaseDates(film.TMDBID)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": film.TMDBID, "error": err}).Errorln("Unable to fetch film release dates from TMDB")
	} else {
		for _, releasesCountry := range releaseDates.Results {
			if releasesCountry.Iso3166_1 == "US" {
				film.Classification = releasesCountry.ReleaseDates[0].Certification
				break
			}
		}
	}

	// Set cast and crew
	credits, err := TMDBClient.GetMovieCredits(film.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": film.TMDBID, "error": err}).Errorln("Unable to fetch film credits from TMDB")
	} else {
		film.Directors = nil
		film.Writers = nil
		film.Characters = nil
		for _, crew := range credits.Crew {
			if crew.Job == "Director" {
				film.Directors = append(film.Directors, crew.ID)
			}
			if crew.Department == "Writing" {
				if !slices.Contains(film.Writers, crew.ID) {
					film.Writers = append(film.Writers, crew.ID)
				}
			}
		}
		for _, cast := range credits.Cast {
			film.Characters = append(film.Characters, model.Character{CharacterName: cast.Character, ActorID: cast.ID})
		}
	}

	film.ProdCountries = nil
	// Set production countries
	for _, country := range details.ProductionCountries {
		film.ProdCountries = append(film.ProdCountries, country.Iso3166_1)
	}
}

// getIMDbRating fetchs rating from IMDbID
func (mw MetadataWrapper) getIMDbRating(imdbId string) string {
	res, err := http.Get(fmt.Sprintf("https://www.imdb.com/title/%s/", imdbId))
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from IMDb")
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from IMDb")
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from IMDb")
		return ""
	}

	return doc.Find("#__next > main > div > section > section > div:nth-child(4) > section > section > div > div > div > div:nth-child(1) > a > div > div > div > div > span").First().Text()
}

// getLetterboxdRating fetchs rating from letterboxd using IMDbID
func (mw MetadataWrapper) getLetterboxdRating(imdbId string) string {
	res, err := http.Get(fmt.Sprintf("https://letterboxd.com/search/films/%s/", imdbId))
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}

	filmUrl, exists := doc.Find("#content > div > div > section > ul > li:nth-child(1) > div").First().Attr("data-target-link")
	if !exists {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}

	res, err = http.Get(fmt.Sprintf("https://letterboxd.com/csi%srating-histogram/", filmUrl))
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}

	return doc.Find("a.display-rating").First().Text()
}

// GetPersonDetails fetches details about a person from TMDB
func (mw MetadataWrapper) GetPersonDetails(personID int64) *model.Person {
	details, err := mw.client.GetPersonDetails(int(personID), nil)
	if err != nil {
		log.WithField("personID", personID).Errorln(err)
		return &model.Person{
			TMDBID: personID,
		}
	}

	return &model.Person{
		TMDBID:   personID,
		Name:     details.Name,
		Photo:    details.ProfilePath,
		Bio:      template.HTML(strings.ReplaceAll(details.Biography, "\n", "<br>")),
		Birthday: details.Birthday,
		Deathday: details.Deathday,
		IMDbID:   details.IMDbID,
	}
}

// GetTMDBIDFromLink returns the TMDB ID from a TMDB, IMDb, or Letterboxd URL
func (mw MetadataWrapper) GetTMDBIDFromLink(inputUrl string) (tmdbID int, err error) {
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return tmdbID, err
	}
	switch urlParsed.Host {
	case "www.themoviedb.org":
		tmdbID, err = mw.getTMDBIDFromTheMovieDB(inputUrl)
	case "www.imdb.com":
		tmdbID, err = mw.getTMDBIDFromIMDB(inputUrl)
	case "letterboxd.com":
		tmdbID, err = mw.getTMDBIDFromLetterboxd(inputUrl)
	default:
		err = errors.New("the host could not be found")
	}

	return tmdbID, err
}

// getTMDBIDFromTheMovieDB returns the TMDB ID from a TMDB URL
func (mw MetadataWrapper) getTMDBIDFromTheMovieDB(inputUrl string) (TMDBID int, err error) {
	err = nil
	// Parse URL
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return TMDBID, err
	}
	var tmdbIDStr string
	// TMDB movie url path should start with /movie/
	if strings.HasPrefix(urlParsed.Path, "/movie/") {
		if strings.HasSuffix(urlParsed.Path, "/") { // this type of link: https://www.themoviedb.org/movie/1817/
			tmdbIDStr = urlParsed.Path[7 : len(urlParsed.Path)-1]
		} else { // this type of link: https://www.themoviedb.org/movie/1817-phone-booth
			tmdbIDStr = strings.Split(urlParsed.Path[7:], "-")[0]
		}
		// TMDB ID shoudl be castable to an integer
		if TMDBID, err = strconv.Atoi(tmdbIDStr); err != nil {
			err = errors.New("could not parse TheMovieDB URL")
		}
	} else {
		err = errors.New("could not parse TheMovieDB URL")
	}
	return TMDBID, err
}

// getTMDBIDFromLetterboxd returns the TMDB ID from a Letterboxd URL
func (mw MetadataWrapper) getTMDBIDFromLetterboxd(inputUrl string) (TMDBID int, err error) {
	// Get the page's HTML
	res, err := http.Get(inputUrl)
	if err != nil {
		log.WithField("url", inputUrl).Errorln("Cannot fetch TMDB ID from Letterboxd")
		return TMDBID, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return TMDBID, errors.New("cannot fetch TMDB ID from Letterboxd")
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return TMDBID, err
	}

	// Get TMDB anchor from the page
	tmdbUrl, exists := doc.Find("a[data-track-action=TMDb]").First().Attr("href")
	if !exists {
		return TMDBID, errors.New("cannot fetch TMDB ID from Letterboxd")
	}
	// Get TMDB ID from its URL
	return mw.getTMDBIDFromTheMovieDB(tmdbUrl)
}

// getTMDBIDFromIMDB returns the TMDB ID from an IMDb URL
func (mw MetadataWrapper) getTMDBIDFromIMDB(inputUrl string) (TMDBID int, err error) {
	// Parse URL
	urlParsed, err := url.Parse(inputUrl)
	if err != nil {
		return TMDBID, err
	}
	// IMDb movie URL path should start with /title/
	if strings.HasPrefix(urlParsed.Path, "/title/") {
		imdbID := urlParsed.Path[7 : len(urlParsed.Path)-1]
		// Get TMDB ID using the TMDB API
		tmdbIDInt64, err := mw.getTMDBIDFromIMDBID(imdbID)
		if err != nil {
			return TMDBID, err
		}
		TMDBID = int(tmdbIDInt64)
	} else {
		err = errors.New("cannot fetch TMDB ID from IMDB")
	}
	return
}

// getTMDBIDFromIMDBID retrieves the TMDB ID from an IMDb ID
func (mw MetadataWrapper) getTMDBIDFromIMDBID(imdbID string) (TMDBID int64, err error) {
	urlOptions := make(map[string]string)
	urlOptions["external_source"] = "imdb_id"
	res, err := TMDBClient.GetFindByID(imdbID, urlOptions)
	if err != nil {
		return TMDBID, err
	}
	TMDBID = res.MovieResults[0].ID
	return
}
