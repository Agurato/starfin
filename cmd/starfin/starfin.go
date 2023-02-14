package main

import (
	"os"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal/server"
	"github.com/Agurato/starfin/internal2/business"
	"github.com/Agurato/starfin/internal2/infrastructure"
	server2 "github.com/Agurato/starfin/internal2/service/server"
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
	// log.SetReportCaller(true)

	db := database.InitMongoDB()
	defer db.Close()

	cache.InitCache()

	go server.InitFileWatching()
	defer server.CloseFileWatching()

	media.InitTMDB()

	server := server.InitServer(db)
	server.Run()
}

func main2() {
	db := infrastructure.NewMongoDB(
		os.Getenv(EnvDBUser),
		os.Getenv(EnvDBPassword),
		os.Getenv(EnvDBURL),
		os.Getenv(EnvDBPort),
		os.Getenv(EnvDBName))

	c := infrastructure.NewCache(os.Getenv(EnvCachePath))
	tmdb, err := infrastructure.NewTMDB()
	if err != nil {
		return
	}

	fmw := business.NewFilmManagerWrapper(db, c, tmdb)
	pmw := business.NewPersonManagerWrapper(db)
	umw := business.NewUserManagerWrapper(db)
	vmw := business.NewVolumeManagerWrapper(db)

	mainHandler := server2.NewMainHandler(umw)
	adminHandler := server2.NewAdminHandler(fmw, umw, vmw)
	filmHandler := server2.NewFilmHandler(fmw, pmw)
	personHandler := server2.NewPersonHandler(pmw, fmw)

	server := server2.NewServer(mainHandler, adminHandler, filmHandler, personHandler)
	server.Run()
}
