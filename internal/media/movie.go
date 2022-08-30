package media

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/agnivade/levenshtein"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/exp/slices"
)

type Movie struct {
	ID          primitive.ObjectID `bson:"_id"`
	VolumeFiles []VolumeFile
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

// NewMovie instantiates a Movie struct implementing
func NewMovie(file string, volumeID primitive.ObjectID, subFiles []string) *Movie {
	filename := filepath.Base(file)
	mediaInfo, err := GetMediaInfo(os.Getenv("MEDIAINFO_PATH"), file)
	if err != nil {
		log.WithField("file", file).Errorln("Could not get media info")
	}
	subtitles := GetExternalSubtitles(file, subFiles)
	movie := Movie{
		ID: primitive.NewObjectID(),
		VolumeFiles: []VolumeFile{{
			Path:         file,
			FromVolume:   volumeID,
			Info:         mediaInfo,
			ExtSubtitles: subtitles,
		}},
	}
	// Split on '.' and ' '
	parts := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '.' || r == ' '
	})
	i := len(parts) - 1

	// Iterate in reverse and stop at first year info
	for ; i >= 0; i-- {
		potentialYear := parts[i]
		if len(potentialYear) == 4 {
			year, err := strconv.Atoi(potentialYear)
			if err == nil {
				movie.ReleaseYear = year
				break
			}
		}
		if len(potentialYear) == 6 && potentialYear[0] == '(' && potentialYear[5] == ')' {
			year, err := strconv.Atoi(potentialYear[1:5])
			if err == nil {
				movie.ReleaseYear = year
				break
			}
		}
	}
	// The movie name should be right before the movie year
	if movie.ReleaseYear > 0 && i >= 0 {
		movie.Name = strings.Join(parts[:i], " ")
	} else {
		movie.Name = strings.Join(parts, " ")
	}

	// Get resolution from name
	resolutionPRegex, _ := regexp.Compile(`^\d\d\d\d?[pP]$`)
	resolutionKRegex, _ := regexp.Compile(`^\d[kK]$`)
	for i := len(parts) - 1; i >= 0; i-- {
		potentialRes := parts[i]
		if resolutionPRegex.MatchString(potentialRes) || resolutionKRegex.MatchString(potentialRes) {
			movie.Resolution = potentialRes
			break
		}
	}
	// If resolution not found, get it from MediaInfo video
	if movie.Resolution == "" {
		movie.Resolution = mediaInfo.Resolution
	}

	return &movie
}

// FetchMediaID fetches media ID from TMDB and stores it
func (m *Movie) FetchTMDBID() error {
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

// FetchDetails fetches media details from TMDB and stores it
func (m *Movie) FetchDetails() {
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
				if !slices.Contains(m.Writers, crew.ID) {
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

func (m Movie) GetCastAndCrewIDs() (ids []int64) {
	for _, cast := range m.Cast {
		ids = append(ids, cast.ActorID)
	}
	ids = append(ids, m.Directors...)
	ids = append(ids, m.Writers...)

	return
}
