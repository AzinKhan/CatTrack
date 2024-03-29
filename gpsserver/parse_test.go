package gpsserver

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseGPS(t *testing.T) {
	testLine := "GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79"
	data, err := ParseGPS(testLine)
	assert.NoError(t, err)
	tempo, _ := time.Parse(layout, "030218224537")
	expected := GPSReading{
		Latitude:  float64(51.417331),
		Longitude: float64(-0.284835),
		Timestamp: tempo,
		Active:    true,
		Speed:     float64(0.29 * 1.852001),
		Bearing:   float64(103.93),
	}
	assert.Equal(t, expected, data)
}

func TestParseCoord(t *testing.T) {
	coord := "5125.0399"
	hemi := "S"
	expected := float64(-51.417331)
	result, err := parseCoord(coord, hemi)
	if err != nil {
		t.Fail()
	}
	if result != expected {
		log.Println("Result:\t", result)
		log.Println("Expected:\t", expected)
		t.Fail()
	}
}

func TestParseGPSReturnsError(t *testing.T) {
	input := "Some wrong input"
	_, err := ParseGPS(input)
	if err == nil {
		t.Fail()
	}
}

func TestParseCoordReturnsError(t *testing.T) {
	wrongCoordInput := "Not a number"
	zero, err := parseCoord(wrongCoordInput, "E")
	if zero != 0 {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
}
