package media

import (
	"os"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal/context"
	tmdb "github.com/cyruzin/golang-tmdb"
)

var (
	// TMDBClient holds the client for tmdb access
	TMDBClient *tmdb.Client
)

// InitTMDB initializes a tmdb client
func InitTMDB() {
	var err error
	TMDBClient, err = tmdb.Init(os.Getenv(context.EnvTMDBAPIKey))
	if err != nil {
		panic(err)
	}
}

const (
	poster       = "poster"
	backdrop     = "backdrop"
	photo        = "photo"
	tmdbImageURL = "https://image.tmdb.org/t/p/"
)

// CachePoster caches a poster from TMDB using its id
func CachePoster(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, poster+key)
	}
	return false, nil
}

// CacheBackdrop caches a backdrop from TMDB using its id
func CacheBackdrop(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W1280+key, backdrop+key)
	}
	return false, nil
}

// CachePhoto caches a person's photo from TMDB using its id
func CachePhoto(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, photo+key)
	}
	return false, nil
}

func GetPoster(key string) ([]byte, error) {
	return cache.GetCachedFile(poster + key)
}

func GetBackdrop(key string) ([]byte, error) {
	return cache.GetCachedFile(backdrop + key)
}

func GetPhoto(key string) ([]byte, error) {
	return cache.GetCachedFile(photo + key)
}

// GetTMDBIDFromIMDBID retrieves the TMDB ID from an IMDb ID
func GetTMDBIDFromIMDBID(imdbID string) (TMDBID int64, err error) {
	urlOptions := make(map[string]string)
	urlOptions["external_source"] = "imdb_id"
	res, err := TMDBClient.GetFindByID(imdbID, urlOptions)
	if err != nil {
		return TMDBID, err
	}
	TMDBID = res.MovieResults[0].ID
	return
}
