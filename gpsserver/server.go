package gpsserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// NewMapHandler returns a handler for the website. It serves the webpage.
func NewMapHandler(webfile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Serving website to", r.RemoteAddr)
		http.ServeFile(w, r, webfile)
	}
}

// NewLocationHandler returns an HTTP handler func for receiving new
// GPS data.
func NewLocationHandler(p *Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.ParseForm()
		u := r.FormValue("Output")
		rawData, err := url.QueryUnescape(u)
		if err != nil {
			log.Println(err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		data, err := ParseGPS(rawData)
		if err != nil {
			log.Printf("Error parsing GPS data: %v", err)
			errstring := fmt.Sprintf("Error parsing gps output: %v", err)
			http.Error(w, errstring, http.StatusBadRequest)
			return
		}
		log.Println("Writing to publisher")
		p.Publish(ctx, data)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Location updated"))
	}
}

var upgrader = websocket.Upgrader{}

// NewSubscriberHandler returns an HTTP handler func for registering a new
// websocket subscriber to listen for new GPS data.
func NewSubscriberHandler(p *Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		// Open websocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error calling upgrader: %v", err)
			status := http.StatusInternalServerError
			http.Error(w, "Could not open websocket", status)
			return
		}
		sw := NewSocketWriter(conn)
		ctx := context.Background()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Run(ctx)
		}()
		removeFunc := p.AddReceiver(ctx, sw)
		defer removeFunc()
		wg.Wait()
	}
}
