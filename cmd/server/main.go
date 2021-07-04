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
	var port, webfile string
	flag.StringVar(&port, "p", "8000", "Port for HTTP server")
	flag.StringVar(&webfile, "f", "", "HTML file to serve")
	flag.Parse()

	router := mux.NewRouter()

	p := gpsserver.NewPublisher()
	ctx := context.Background()
	p.AddReceiver(ctx, &gpsserver.Logger{})
	go p.Run(ctx)
	router.HandleFunc("/map", gpsserver.NewMapHandler(webfile))
	router.HandleFunc("/marker", gpsserver.NewLocationHandler(p))
	router.HandleFunc("/subscribe", gpsserver.NewSubscriberHandler(p))
	address := fmt.Sprintf("0.0.0.0:%v", port)
	server := &http.Server{
		Addr:         address,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  3 * time.Second,
		Handler:      router,
	}
	log.Println("Starting HTTP server on port", port)
	log.Fatal(server.ListenAndServe())
}
