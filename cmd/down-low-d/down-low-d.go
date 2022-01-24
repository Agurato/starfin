package main

import (
	"github.com/Agurato/down-low-d/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	mongoClient := server.InitMongo()
	defer mongoClient.Disconnect(server.MongoCtx)

	server.InitTMDB()

	server := server.InitServer()
	server.Run() // default port is :8080
}
