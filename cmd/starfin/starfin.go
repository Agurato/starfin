package main

import (
	"os"

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

	mongoClient := server.InitMongo()
	defer mongoClient.Disconnect(server.MongoCtx)

	media.InitTMDB()

	server := server.InitServer()
	server.Run() // default port is :8080
}
