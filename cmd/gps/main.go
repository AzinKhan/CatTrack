package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AzinKhan/CatTrack/gps"
)

func main() {
	flag.Parse()
	gpschan := make(chan string)
	port, err := gps.InitializePort(gps.UARTPort)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	go gps.Readgps(port, gpschan)
	for {
		gpsOut := <-gpschan
		if strings.Contains(gpsOut, "GPRMC") {
			urlData := url.Values{}
			urlData.Add("Output", gpsOut)
			log.Printf("Read GPS output: %+v", gpsOut)
			resp, err := client.PostForm(gps.ServerIP+"/marker", urlData)
			if err != nil {
				log.Println(err)
				// Continue if there is an error
				// This avoids null pointer panics when trying to close the response body below
				// TODO: Add checks to see if the resp exists before continuing
				continue
			}
			resp.Body.Close()

		}
	}
}
