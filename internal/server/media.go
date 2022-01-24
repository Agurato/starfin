package server

import (
	"fmt"
	"os"
	"strings"

	tmdb "github.com/cyruzin/golang-tmdb"
	"go.mongodb.org/mongo-driver/bson"
)

// MediaInfo contains the information about one item from the search results
type MediaInfo struct {
	ID        int64
	MediaType string
	MediaURL  string
	Title     string
	Date      string
	Language  string
	Poster    string
	Backdrop  string

	Action string
}

// Media actions
const (
	MediaActionAdd    string = "add"
	MediaActionRemove string = "remove"

	MediaMovie string = "movie"
	MediaTV    string = "tv"
)

var (
	// TMDBClient holds the client for tmdb access
	TMDBClient *tmdb.Client
)

// InitTMDB initializes a tmdb client
func InitTMDB() {
	var err error
	TMDBClient, err = tmdb.Init(os.Getenv(EnvTMDBAPIKey))
	if err != nil {
		panic(err)
	}
}

// MediaSearchMulti returns an array of media informations corresponding to search query on TMDB
func MediaSearchMulti(searchQuery string, user User) ([]MediaInfo, error) {
	userColl := mongoDb.Collection("users")
	var searchResults []MediaInfo

	tmdbSearchRes, err := TMDBClient.GetSearchMulti(searchQuery, nil)
	if err != nil {
		return searchResults, err
	}

	for _, res := range tmdbSearchRes.Results {
		action := MediaActionAdd
		var userDB User
		if err := userColl.FindOne(MongoCtx, bson.M{"_id": user.ID}).Decode(&userDB); err != nil {
			// TODO
		}

		switch res.MediaType {
		case "movie":
			title := res.Title
			if res.OriginalLanguage == "fr" {
				title = res.OriginalTitle
			} else if res.OriginalLanguage != "en" && res.Title != res.OriginalTitle {
				title = fmt.Sprintf("%s (%s)", res.Title, res.OriginalTitle)
			}
			posterPath := ""
			if len(res.PosterPath) > 0 {
				posterPath = tmdb.GetImageURL(res.PosterPath, "original")
			}
			searchResults = append(searchResults, MediaInfo{
				ID:        res.ID,
				MediaType: "Movie",
				MediaURL:  fmt.Sprintf("/movie/%d", res.ID),
				Title:     title,
				Date:      strings.Split(res.ReleaseDate, "-")[0],
				Language:  res.OriginalLanguage,
				Poster:    posterPath,
				Action:    action,
			})
		case "tv":
			title := res.Name
			if res.OriginalLanguage == "fr" {
				title = res.OriginalName
			} else if res.OriginalLanguage != "en" && res.Name != res.OriginalName {
				title = fmt.Sprintf("%s (%s)", res.Name, res.OriginalName)
			}
			posterPath := ""
			if len(res.PosterPath) > 0 {
				posterPath = tmdb.GetImageURL(res.PosterPath, "original")
			}
			searchResults = append(searchResults, MediaInfo{
				ID:        res.ID,
				MediaType: "TV",
				MediaURL:  fmt.Sprintf("/tv/%d", res.ID),
				Title:     title,
				Date:      strings.Split(res.FirstAirDate, "-")[0],
				Language:  res.OriginalLanguage,
				Poster:    posterPath,
				Action:    action,
			})
		}
	}

	return searchResults, nil
}

// MediaMovieDetails fetches movie details from TMDB
func MediaMovieDetails(tmdbID int, inUserQueries bool) (MediaInfo, error) {
	info := MediaInfo{}

	details, err := TMDBClient.GetMovieDetails(tmdbID, nil)
	if err != nil {
		return info, err
	}
	title := details.Title
	if details.OriginalLanguage == "fr" {
		title = details.OriginalTitle
	} else if details.OriginalLanguage != "en" && details.Title != details.OriginalTitle {
		title = fmt.Sprintf("%s (%s)", details.Title, details.OriginalTitle)
	}
	posterPath := ""
	if len(details.PosterPath) > 0 {
		posterPath = tmdb.GetImageURL(details.PosterPath, "original")
	}
	backdropPath := ""
	if len(details.PosterPath) > 0 {
		backdropPath = tmdb.GetImageURL(details.BackdropPath, "original")
	}

	action := MediaActionAdd
	if inUserQueries {
		action = MediaActionRemove
	}

	return MediaInfo{
		ID:        details.ID,
		MediaType: "Movie",
		Title:     title,
		Date:      strings.Split(details.ReleaseDate, "-")[0],
		Language:  details.OriginalLanguage,
		Poster:    posterPath,
		Backdrop:  backdropPath,
		Action:    action,
	}, nil
}
