package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AzinKhan/CatTrack/gps"
	"github.com/tarm/serial"
)

func main() {
	var UARTPort, addr string
	flag.StringVar(&UARTPort, "port", "/dev/ttyS0", "Serial port for connection")
	flag.StringVar(&addr, "s", "http://localhost:8000", "Address of remote server")
	flag.Parse()
	config := &serial.Config{
		Name: UARTPort,
		Baud: 9600,
	}
	log.Println("Opening port", UARTPort)
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	reader := gps.NewReader(port)

	messages := reader.ReadGPS()

	for gpsOut := range messages {
		if !strings.Contains(gpsOut, "GPRMC") {
			continue
		}
		urlData := url.Values{}
		urlData.Add("Output", gpsOut)
		log.Printf("Read GPS output: %+v", gpsOut)
		resp, err := client.PostForm(addr+"/marker", urlData)
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
