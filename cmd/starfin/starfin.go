package main

import (
	"os"

	"github.com/Agurato/starfin/internal/database"
	"github.com/Agurato/starfin/internal/media"
	"github.com/Agurato/starfin/internal/server"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()

	log.SetOutput(os.Stdout)
	// TODO: Set level via environment variable
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	db := database.InitMongoDB()
	defer db.Close()

	go server.InitFileWatching()
	defer server.CloseFileWatching()

	media.InitTMDB()

	server := server.InitServer(db)
	server.Run() // default port is :8080
}
