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

func CachePoster(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, poster+key)
	}
	return false, nil
}

func CacheBackdrop(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W1280+key, backdrop+key)
	}
	return false, nil
}

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
