package main

import (
	"fmt"

	"github.com/Agurato/down-low-d/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	server.InitTMDB()
	fmt.Println(server.TMDBSearchMovie("1917", 2019))
	fmt.Println(server.TMDBSearchMovie("1917", 0))
}
