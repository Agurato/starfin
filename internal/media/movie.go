package media

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	tmdb "github.com/cyruzin/golang-tmdb"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Movie struct {
	ID          primitive.ObjectID `bson:"_id"`
	Paths       []VolumeFile
	Name        string // Name fetched from filename
	Resolution  string // Resolution fetched from filename
	ReleaseYear int    // Release year fetched from filename
	TMDBID      int
	IMDbID      string

	// Fetched from online sources. Only these variables will be used by the template
	Title            string
	OriginalTitle    string
	Year             string
	Runtime          string
	Tagline          string
	Overview         string
	TinyPosterPath   string
	PosterPath       string
	BackdropPath     string
	Classification   string
	IMDbRating       string
	LetterboxdRating string
	Genres           []string
	Directors        []string
	Writers          []string
	Cast             []string
	ProdCountries    []string
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

	mostPopular := float32(0)
	for _, res := range tmdbSearchRes.Results {
		if res.Popularity > mostPopular {
			// Levenshtein distance so that the name corresponds at least a little bit
			if levenshtein.ComputeDistance(m.Name, res.Title) < len(m.Name)/3 || mostPopular == 0 {
				m.TMDBID = int(res.ID)
				mostPopular = res.Popularity
			}
		}
	}
	return nil
}

func (m *Movie) FetchMediaDetails() {
	// Get details
	details, err := TMDBClient.GetMovieDetails(m.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie details from TMDB")
	}
	m.IMDbID = details.IMDbID
	m.Title = details.Title
	m.OriginalTitle = details.OriginalTitle
	m.Year = details.ReleaseDate[:4]
	m.Runtime = strconv.Itoa(details.Runtime)
	m.Tagline = details.Tagline
	m.Overview = details.Overview
	if details.PosterPath != "" {
		m.TinyPosterPath = tmdb.GetImageURL(details.PosterPath, "w154")
		m.PosterPath = tmdb.GetImageURL(details.PosterPath, "w342")
	}
	m.BackdropPath = tmdb.GetImageURL(details.BackdropPath, "w1280")
	m.IMDbRating = GetIMDbRating(m.IMDbID)
	m.LetterboxdRating = GetLetterboxdRating(m.IMDbID)

	// Set genres
	for _, genre := range details.Genres {
		m.Genres = append(m.Genres, genre.Name)
	}

	// Set classification
	releaseDates, err := TMDBClient.GetMovieReleaseDates(m.TMDBID)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie release dates from TMDB")
	} else {
		for _, releasesCountry := range releaseDates.Results {
			if releasesCountry.Iso3166_1 == "US" {
				m.Classification = releasesCountry.ReleaseDates[0].Certification
				break
			}
		}
	}

	// Set crew
	credits, err := TMDBClient.GetMovieCredits(m.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie credits from TMDB")
	} else {
		for _, crew := range credits.Crew {
			if crew.Job == "Director" {
				m.Directors = append(m.Directors, crew.Name)
			}
			if crew.Department == "Writing" {
				m.Writers = append(m.Writers, crew.Name)
			}
		}
		for _, cast := range credits.Cast {
			m.Cast = append(m.Cast, cast.Name)
		}
	}

	// Set production countries
	for _, country := range details.ProductionCountries {
		m.ProdCountries = append(m.ProdCountries, country.Iso3166_1)
	}
}

func (m Movie) GetTMDBID() int {
	return m.TMDBID
}

// GetIMDbRating fetchs rating from IMDbID
func GetIMDbRating(imdbId string) string {
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

// GetLetterboxdRating fetchs rating from letterboxd using IMDbID
func GetLetterboxdRating(imdbId string) string {
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

	movieUrl, exists := doc.Find("#content > div > div > section > ul > li:nth-child(1) > div").First().Attr("data-target-link")
	if !exists {
		log.WithField("imdb_id", imdbId).Errorln("Cannot fetch rating from Letterboxd")
		return ""
	}

	res, err = http.Get(fmt.Sprintf("https://letterboxd.com/csi%srating-histogram/", movieUrl))
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
