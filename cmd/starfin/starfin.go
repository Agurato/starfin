package main

import (
	"os"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal/server"
	"github.com/Agurato/starfin/internal2/infrastructure"
	server2 "github.com/Agurato/starfin/internal2/service/server"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
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
		os.Getenv(ctx.EnvDBUser),
		os.Getenv(ctx.EnvDBPassword),
		os.Getenv(ctx.EnvDBURL),
		os.Getenv(ctx.EnvDBPort),
		os.Getenv(ctx.EnvDBName))

	mainHandler := server2.NewMainHandler(db)
	adminHandler := server2.NewAdminHandler(db)
	filmHandler := server2.NewFilmHandler(db)
	personHandler := server2.NewPersonHandler(db)

	server := server2.NewServer(mainHandler, adminHandler, filmHandler, personHandler)
	server.Run()
}
