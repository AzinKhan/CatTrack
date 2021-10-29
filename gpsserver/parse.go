package gpsserver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

const layout string = "020106150405"

const unit float64 = 0.0000005

// GPS gives speed in knots so convert to km/h
const knotRatio float64 = 1.852001

// GPSReading is a container for all of the information contained in the
// output from gps
type GPSReading struct {
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

// ParseGPS takes the output from gps and populates a GPSReading struct with
// the relevant values.
func ParseGPS(outputline string) (*GPSReading, error) {
	// The data come as one string delineated by commas

	// ----------TIMESTAMP--------------
	timeRegex := regexp.MustCompile("\\d\\d\\d\\d\\d\\d")
	dateTime := timeRegex.FindAllString(outputline, -1)
	if len(dateTime) != 2 {
		return nil, errors.New("Could not parse timestamp")
	}
	timestamp, err := time.Parse(layout, (dateTime[1] + dateTime[0]))
	if err != nil {
		return nil, err
	}
	// Replace matches with empty strings to prevent matching
	// again on regexs below
	outputline = timeRegex.ReplaceAllString(outputline, "")

	// ----------ACTIVE----------------
	activeRegex := regexp.MustCompile("A,")
	active := activeRegex.MatchString(outputline)
	if !active {
		return nil, errors.New("No fix yet")
	}

	// ----------COORDINATES-----------
	coordRegex := regexp.MustCompile("(\\d+.\\d+).([NESW])")
	coords := coordRegex.FindAllStringSubmatch(outputline, -1)
	if len(coords) != 2 || len(coords[0]) != 3 && len(coords[1]) != 3 {
		return nil, errors.New("Unexpected coordinate format")
	}
	latitude, err := ParseCoord(coords[0][1], coords[0][2])
	if err != nil {
		return nil, err
	}
	longitude, err := ParseCoord(coords[1][1], coords[1][2])
	if err != nil {
		return nil, err
	}
	outputline = coordRegex.ReplaceAllString(outputline, "")

	// ----------VELOCITY-------------
	velocityRegex := regexp.MustCompile("\\d+\\.\\d+")
	velocity := velocityRegex.FindAllString(outputline, -1)
	if len(velocity) != 2 {
		return nil, errors.New("could not parse velocity")
	}

	knotspeed, err := strconv.ParseFloat(velocity[0], 64)
	if err != nil {
		return nil, err
	}
	// Convert to km/h
	speed := knotspeed * knotRatio
	bearing, err := strconv.ParseFloat(velocity[1], 64)
	if err != nil {
		return nil, err
	}

	return &GPSReading{
		Timestamp: timestamp,
		Latitude:  latitude,
		Longitude: longitude,
		Speed:     speed,
		Bearing:   bearing,
		Active:    active,
	}, nil
}
