package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AzinKhan/CatTrack/gpsserver"

	"github.com/gorilla/mux"
)

func main() {
	flag.Parse()
	router := mux.NewRouter()

	p := gpsserver.NewPublisher()
	ctx := context.Background()
	p.AddReceiver(ctx, &gpsserver.Logger{})
	go p.Run(ctx)
	router.HandleFunc("/map", gpsserver.SendMap)
	router.HandleFunc("/marker", gpsserver.NewLocationHandler(p))
	router.HandleFunc("/subscribe", gpsserver.NewSubscriberHandler(p))
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
