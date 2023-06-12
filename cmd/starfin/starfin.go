package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Agurato/starfin/internal/business"
	"github.com/Agurato/starfin/internal/infrastructure"
	"github.com/Agurato/starfin/internal/model"
	"github.com/Agurato/starfin/internal/service/server"
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
	EnvItemsPerPage = "ITEMS_PER_PAGE"

	EnvEnableRarbg     = "ENABLE_RARBG"
	EnvTorznabAPIKey   = "TORZNAB_API_KEY"
	EnvRarbgSqliteFile = "RARBG_SQLITE_FILE"
)

func main() {
	err := initApp()
	if err != nil {
		log.WithError(err).Errorln("Error during initialization")
	}
}

func initApp() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

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
		return err
	}

	enableRarbg, err := strconv.ParseBool(os.Getenv(EnvEnableRarbg))
	if err != nil {
		return fmt.Errorf("error parsing env var %q: %w", EnvEnableRarbg, err)
	}

	filterer := business.NewFilterer()
	fm := business.NewFilmManager(db, c, metadata, filterer)
	filterer.AddFilms(fm.GetFilms())

	fw := business.NewFileWatcher(db, fm, metadata)
	go fw.Run()

	pm := business.NewPersonManager(db)
	um := business.NewUserManager(db)
	vm := business.NewVolumeManager(db, fw, fm, metadata)

	itemsPerPage, err := strconv.ParseInt(os.Getenv(EnvItemsPerPage), 10, 64)
	if err != nil {
		return fmt.Errorf("error getting ITEMS_PER_PAGE: %w", err)
	}
	fp := business.NewPaginater[model.Film](itemsPerPage)
	pp := business.NewPaginater[model.Person](itemsPerPage)

	mainHandler := server.NewMainHandler(c, um)
	adminHandler := server.NewAdminHandler(fm, um, vm)
	filmHandler := server.NewFilmHandler(fm, pm, filterer, fp)
	personHandler := server.NewPersonHandler(pm, fm, pp)

	var rarbgHandler *server.RarbgHandler = nil
	if enableRarbg {
		db.InitRarbg("rarbg")
		rarbgHandler = server.NewRarbgHandler(db, os.Getenv(EnvTorznabAPIKey))
	}

	srv := server.NewServer(
		os.Getenv(EnvCookieSecret),
		mainHandler,
		adminHandler,
		filmHandler,
		personHandler,
		rarbgHandler,
		db)
	err = srv.Run()
	if err != nil {
		return err
	}

	return nil
}
