package media

import (
	"os"

	tmdb "github.com/cyruzin/golang-tmdb"
)

var (
	// TMDBClient holds the client for tmdb access
	TMDBClient *tmdb.Client
)

// InitTMDB initializes a tmdb client
func InitTMDB() {
	var err error
	TMDBClient, err = tmdb.Init(os.Getenv("TMDB_API_KEY"))
	if err != nil {
		panic(err)
	}
}
