package media

import (
	"os"

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
