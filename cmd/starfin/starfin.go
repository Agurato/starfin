package main

import (
	"os"
	"strconv"

	"github.com/Agurato/starfin/internal/business"
	"github.com/Agurato/starfin/internal/infrastructure"
	"github.com/Agurato/starfin/internal/model"
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
	EnvItemsPerPage = "ITEMS_PER_PAGE"
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

	itemsPerPage, err := strconv.ParseInt(os.Getenv(EnvItemsPerPage), 10, 64)
	if err != nil {
		log.Fatalf("error getting ITEMS_PER_PAGE: %v", err)
		return
	}
	fp := business.NewPaginaterWrapper[model.Film](itemsPerPage)
	pp := business.NewPaginaterWrapper[model.Person](itemsPerPage)

	mainHandler := server2.NewMainHandler(c, umw)
	adminHandler := server2.NewAdminHandler(fmw, umw, vmw)
	filmHandler := server2.NewFilmHandler(fmw, pmw, filterer, fp)
	personHandler := server2.NewPersonHandler(pmw, fmw, pp)

	server := server2.NewServer(
		os.Getenv(EnvCookieSecret),
		mainHandler,
		adminHandler,
		filmHandler,
		personHandler,
		db)
	server.Run()
}
