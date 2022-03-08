package media

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Agurato/starfin/internal/utilities"
	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
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
	PosterPath       string
	BackdropPath     string
	Classification   string
	IMDbRating       string
	LetterboxdRating string
	Genres           []string
	Directors        []int64
	Writers          []int64
	Cast             []Cast
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

// FetchMediaDetails fetches media details from TMDB and stores it in the Movie structure
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
	m.PosterPath = details.PosterPath
	m.BackdropPath = details.BackdropPath
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

	// Set cast and crew
	credits, err := TMDBClient.GetMovieCredits(m.TMDBID, nil)
	if err != nil {
		log.WithFields(log.Fields{"tmdbID": m.TMDBID, "error": err}).Errorln("Unable to fetch movie credits from TMDB")
	} else {
		for _, crew := range credits.Crew {
			if crew.Job == "Director" {
				m.Directors = append(m.Directors, crew.ID)
			}
			if crew.Department == "Writing" {
				if !utilities.Int64SliceContains(m.Writers, crew.ID) {
					m.Writers = append(m.Writers, crew.ID)
				}
			}
		}
		for _, cast := range credits.Cast {
			m.Cast = append(m.Cast, Cast{Character: cast.Character, ActorID: cast.ID})
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

func (m Movie) GetCastAndCrewIDs() (ids []int64) {
	for _, cast := range m.Cast {
		ids = append(ids, cast.ActorID)
	}
	ids = append(ids, m.Directors...)
	ids = append(ids, m.Writers...)

	return
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

// SearchMovies returns a sublist of movies containing the search terms
// Searches in the title and original title (case-insensitive)
// Searches movies from specific year (indicated by "y:XXXX" as the last part of the search)
func SearchMovies(search string, movies []Movie) ([]Movie, string, int) {
	search = strings.Trim(search, " ")
	searchSplit := strings.Split(search, " ")
	yearRegex := regexp.MustCompile(`^y:\d{4}$`)
	specialChars := regexp.MustCompile("[.,\\/#!$%\\^&\\*;:{}=\\-_`~()%\\s\\\\]")
	lastSearchIdx := len(searchSplit) - 1
	var (
		searchYear     int
		filteredMovies []Movie
	)

	// If there's a year in last part of search term, return false if the movie is not from that year
	if yearRegex.MatchString(searchSplit[lastSearchIdx]) {
		searchYear, _ = strconv.Atoi(searchSplit[lastSearchIdx][2:])
		searchSplit = searchSplit[:lastSearchIdx]
	}

	search = strings.Join(searchSplit, "")
	search = specialChars.ReplaceAllString(strings.ToLower(search), "")
	for _, m := range movies {
		if searchYear != 0 && m.ReleaseYear != searchYear {
			continue
		}
		title := specialChars.ReplaceAllString(strings.ToLower(m.Title), "")
		originalTitle := specialChars.ReplaceAllString(strings.ToLower(m.OriginalTitle), "")
		if strings.Contains(title, search) || strings.Contains(originalTitle, search) {
			filteredMovies = append(filteredMovies, m)
		}
	}

	return filteredMovies, strings.Join(searchSplit, " "), searchYear
}
