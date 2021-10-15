package gpsserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const layout string = "020106150405"

const unit float64 = 0.0000005

// GPS gives speed in knots so convert to km/h
const knotRatio float64 = 1.852001

// GPSdata is a container for all of the information contained in the
// output from gps
type GPSdata struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
	Active    bool
	Speed     float64
	Bearing   float64
}

// Round implements a rounding feature not available in Go 1.9
func Round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}

// ParseCoord converts the co-ordinates from the gps module to those
// understandable by GoogleMaps. GPS gives DDMM.MMMM format output
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

// ParseGPS takes the output from gps and populates a GPSdata struct with
// the relevant values.
func (data *GPSdata) ParseGPS(outputline string) error {
	// The data come as one string delineated by commas
	timeRegex := regexp.MustCompile("\\d\\d\\d\\d\\d\\d")
	dateTime := timeRegex.FindAllString(outputline, -1)
	if len(dateTime) != 2 {
		return errors.New("Could not parse timestamp")
	}
	var err error
	data.Timestamp, err = time.Parse(layout, (dateTime[1] + dateTime[0]))
	if err != nil {
		return err
	}
	// Replace matches with empty strings to prevent matching
	// again on regexs below
	outputline = timeRegex.ReplaceAllString(outputline, "")

	activeRegex := regexp.MustCompile("A,")
	data.Active = activeRegex.MatchString(outputline)
	if !data.Active {
		return errors.New("No fix yet")
	}

	coordRegex := regexp.MustCompile("(\\d+.\\d+).([NESW])")
	coords := coordRegex.FindAllStringSubmatch(outputline, -1)
	if len(coords) != 2 || len(coords[0]) != 3 && len(coords[1]) != 3 {
		return errors.New("Unexpected coordinate format")
	}
	data.Latitude, err = ParseCoord(coords[0][1], coords[0][2])
	if err != nil {
		return err
	}
	data.Longitude, err = ParseCoord(coords[1][1], coords[1][2])
	if err != nil {
		return err
	}
	outputline = coordRegex.ReplaceAllString(outputline, "")

	velocityRegex := regexp.MustCompile("\\d+\\.\\d+")
	velocity := velocityRegex.FindAllString(outputline, -1)
	if len(velocity) != 2 {
		return errors.New("could not parse velocity")
	}

	knotspeed, err := strconv.ParseFloat(velocity[0], 64)
	if err != nil {
		return err
	}
	// Convert to km/h
	data.Speed = knotspeed * knotRatio
	data.Bearing, err = strconv.ParseFloat(velocity[1], 64)
	if err != nil {
		return err
	}
	return nil
}

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
		data := &GPSdata{}
		err = data.ParseGPS(rawData)
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
