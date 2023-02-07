package infrastructure

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

type TMDB struct {
	client *tmdb.Client
}

// NewTMDB initializes a tmdb client
func NewTMDB() (*TMDB, error) {
	client, err := tmdb.Init(os.Getenv(context.EnvTMDBAPIKey))
	if err != nil {
		return nil, err
	}
	return &TMDB{
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
func (t TMDB) GetPosterLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

// GetBackdropLink caches a backdrop from TMDB using its id
func (t TMDB) GetBackdropLink(key string) string {
	return tmdbImageURL + tmdb.W1280 + key
}

// GetPhotoLink caches a person's photo from TMDB using its id
func (t TMDB) GetPhotoLink(key string) string {
	return tmdbImageURL + tmdb.W342 + key
}

// CachePoster caches a poster from TMDB using its id
func (t TMDB) CachePoster(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, poster+key)
	}
	return false, nil
}

// CacheBackdrop caches a backdrop from TMDB using its id
func (t TMDB) CacheBackdrop(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W1280+key, backdrop+key)
	}
	return false, nil
}

// CachePhoto caches a person's photo from TMDB using its id
func (t TMDB) CachePhoto(key string) (bool, error) {
	if key != "" {
		return cache.CacheFile(tmdbImageURL+tmdb.W342+key, photo+key)
	}
	return false, nil
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
