package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var port, webfile string

const layout string = "020106150405.000"

// GPS gives speed in knots so convert to km/h
const knotRatio float64 = 1.852001

func init() {
	flag.StringVar(&port, "p", "8000", "Port for HTTP server")
	flag.StringVar(&webfile, "f", "", "HTML file to serve")
}

type GPSdata struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
	Active    bool
	Speed     float64
	Angle     float64
	mutex     sync.Mutex
}

func ParseCoord(coord, hemi string) (float64, error) {
	coordinate, err := strconv.ParseFloat(coord, 64)
	if err != nil {
		return 0, err
	}
	switch hemi {
	case "W":
		coordinate = -1.0 * coordinate
	case "S":
		coordinate = -1.0 * coordinate
	}
	return coordinate, nil
}

func (data *GPSdata) ParseGPS(outputline string) error {
	splitz := strings.Split(outputline, ",")
	if len(splitz) != 13 {
		return fmt.Errorf("Not an expected input:\n", splitz)
	}
	var err error
	data.Timestamp, err = time.Parse(layout, (splitz[9] + splitz[1]))
	if err != nil {
		return err
	}
	if splitz[2] == "A" {
		data.Active = true
	} else {
		return fmt.Errorf("No fix yet")
	}
	data.Latitude, err = ParseCoord(splitz[3], splitz[4])
	if err != nil {
		return err
	}
	data.Longitude, err = ParseCoord(splitz[5], splitz[6])
	if err != nil {
		return err
	}
	knotspeed, err := strconv.ParseFloat(splitz[7], 64)
	// Convert to km/h
	data.Speed = knotspeed * knotRatio
	if err != nil {
		return err
	}
	data.Angle, err = strconv.ParseFloat(splitz[8], 64)
	if err != nil {
		return err
	}
	return nil
}

func SendMap(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving website to", r.RemoteAddr)
	http.ServeFile(w, r, webfile)
}

func UpdateMarker(data *GPSdata) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			data.mutex.Lock()
			r.ParseForm()
			u := r.FormValue("Output")
			rawData, err := url.QueryUnescape(u)
			if err != nil {
				log.Println(err)
			}
			err = data.ParseGPS(rawData)
			if err != nil {
				w.WriteHeader(400)
				errstring := fmt.Sprintf("Error parsing gps output: %v", err)
				w.Write([]byte(errstring))
			} else {
				w.WriteHeader(200)
				w.Write([]byte("Location updated"))
			}
			data.mutex.Unlock()
		case "GET":
			data.mutex.Lock()
			dataBytes, err := json.Marshal(data)
			data.mutex.Unlock()
			if err != nil {
				w.WriteHeader(404)
				errstring := fmt.Sprintf("Error retrieving location: %v", err)
				w.Write([]byte(errstring))
			}
			w.WriteHeader(200)
			w.Write(dataBytes)
		default:
			w.WriteHeader(400)
			errstring := fmt.Sprintf("%v method not supported", r.Method)
			w.Write([]byte(errstring))
		}
	}
}

func main() {
	flag.Parse()
	server := mux.NewRouter()
	var latestData GPSdata
	//server := http.NewServeMux()
	server.HandleFunc("/map", SendMap)
	// This receives the post requests
	server.HandleFunc("/marker", UpdateMarker(&latestData))
	log.Println("Starting HTTP server on port", port)
	address := fmt.Sprintf("0.0.0.0:%v", port)
	log.Fatal(http.ListenAndServe(address, server))
}
