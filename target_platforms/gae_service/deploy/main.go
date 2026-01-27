package main

import (
	"log"
	"net/http"
	"os"

	"github.com/darianmavgo/flight3/internal/flight"
)

func main() {
	// Set up Flight3
	go flight.Flight()

	// App Engine requires listening on PORT env var
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Flight3 on App Engine, port %s", port)
	
	// Keep the main function alive
	select {}
}
