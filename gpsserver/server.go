package gpsserver

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Port is the listening port for the server
var Port, webfile string

const layout string = "020106150405.000"

const unit float64 = 0.0000005

// GPS gives speed in knots so convert to km/h
const knotRatio float64 = 1.852001

func init() {
	flag.StringVar(&Port, "p", "8000", "Port for HTTP server")
	flag.StringVar(&webfile, "f", "", "HTML file to serve")
}

type GPSdata struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
	Active    bool
	Speed     float64
	Bearing   float64
	mutex     sync.Mutex
}

// Round implements a rounding feature not available in Go 1.9
func Round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}

// GPS gives DDMM.MMMM format output
func ParseCoord(coord, hemi string) (float64, error) {
	minuteString := coord[2:]
	degreeString := coord[:2]
	minutes, err := strconv.ParseFloat(minuteString, 64)
	if err != nil {
		return 0, err
	}
	degrees, err := strconv.ParseFloat(degreeString, 64)
	// Convert to decimal
	coordinate := degrees + Round((minutes/60.0), unit)
	coordinate = Round(coordinate, unit)
	coordinateString := fmt.Sprintf("%.6f", coordinate)
	coordinate, err = strconv.ParseFloat(coordinateString, 64)
	if err != nil {
		return 0, err
	}
	// Change sign of co-ordinate if either in West or South hemisphere
	switch hemi {
	case "W":
		coordinate = -1.0 * coordinate
	case "S":
		coordinate = -1.0 * coordinate
	}
	return coordinate, nil
}

func (data *GPSdata) ParseGPS(outputline string) error {
	// The data come as one string delineated by commas
	splitz := strings.Split(outputline, ",")
	if len(splitz) != 13 {
		return fmt.Errorf("not an expected input %v", splitz)
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
	if err != nil {
		return err
	}
	// Convert to km/h
	data.Speed = knotspeed * knotRatio
	data.Bearing, err = strconv.ParseFloat(splitz[8], 64)
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
			log.Println("Handling POST request from", r.RemoteAddr)
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
			r.Close = true
			data.mutex.Unlock()
		case "GET":
			log.Println("Handling GET request from", r.RemoteAddr)
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
			r.Close = true
		default:
			w.WriteHeader(400)
			errstring := fmt.Sprintf("%v method not supported", r.Method)
			w.Write([]byte(errstring))
			r.Close = true
		}
	}
}
