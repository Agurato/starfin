package main

import (
	"os"

	"github.com/Agurato/starfin/internal/business"
	"github.com/Agurato/starfin/internal/infrastructure"
	server2 "github.com/Agurato/starfin/internal/service/server"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// Environment variables names
const (
	EnvCookieSecret = "COOKIE_SECRET"
	EnvDBURL        = "DB_URL"
	EnvDBPort       = "DB_PORT"
	EnvDBName       = "DB_NAME"
	EnvDBUser       = "DB_USER"
	EnvDBPassword   = "DB_PASSWORD"
	EnvTMDBAPIKey   = "TMDB_API_KEY" // This may be configurable via admin panel in the future
	EnvCachePath    = "CACHE_PATH"
)

func main() {
	godotenv.Load()

	log.SetOutput(os.Stdout)
	// TODO: Set level via environment variable
	log.SetLevel(log.DebugLevel)

	db := infrastructure.NewMongoDB(
		os.Getenv(EnvDBUser),
		os.Getenv(EnvDBPassword),
		os.Getenv(EnvDBURL),
		os.Getenv(EnvDBPort),
		os.Getenv(EnvDBName))

	c := infrastructure.NewCache(os.Getenv(EnvCachePath))
	metadata, err := infrastructure.NewMetadataWrapper(os.Getenv(EnvTMDBAPIKey))
	if err != nil {
		return
	}

	filterer := business.NewFiltererWrapper()
	fmw := business.NewFilmManagerWrapper(db, c, metadata, filterer)
	filterer.AddFilms(fmw.GetFilms())

	fw := business.NewFileWatcher(db, fmw, metadata)
	go fw.Run()

	pmw := business.NewPersonManagerWrapper(db)
	umw := business.NewUserManagerWrapper(db)
	vmw := business.NewVolumeManagerWrapper(db, fw, fmw, metadata)

	mainHandler := server2.NewMainHandler(c, umw)
	adminHandler := server2.NewAdminHandler(fmw, umw, vmw)
	filmHandler := server2.NewFilmHandler(fmw, pmw, filterer)
	personHandler := server2.NewPersonHandler(pmw, fmw)

	server := server2.NewServer(
		os.Getenv(EnvCookieSecret),
		mainHandler,
		adminHandler,
		filmHandler,
		personHandler,
		db)
	server.Run()
}
