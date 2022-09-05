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

func CachePoster(key string) error {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, poster+key)
	}
	return nil
}

func CacheBackdrop(key string) error {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W1280+key, backdrop+key)
	}
	return nil
}

func CachePhoto(key string) error {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, photo+key)
	}
	return nil
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
