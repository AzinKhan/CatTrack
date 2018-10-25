package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"cattrack/gpsserver"

	"github.com/gorilla/mux"
)

func main() {
	flag.Parse()
	router := mux.NewRouter()
	var latestData gpsserver.GPSdata
	router.HandleFunc("/map", gpsserver.SendMap)
	// This receives the post requests
	router.HandleFunc("/marker", gpsserver.UpdateMarker(&latestData))
	address := fmt.Sprintf("0.0.0.0:%v", gpsserver.Port)
	server := &http.Server{
		Addr:         address,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  3 * time.Second,
		Handler:      router,
	}
	log.Println("Starting HTTP server on port", gpsserver.Port)
	log.Fatal(server.ListenAndServe())
}
