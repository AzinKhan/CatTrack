package gpsserver

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseGPS(t *testing.T) {
	var fakeGPSdata GPSdata
	testLine := "GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79"
	err := fakeGPSdata.ParseGPS(testLine)
	assert.NoError(t, err)
	tempo, _ := time.Parse(layout, "030218224537")
	expected := GPSdata{
		Latitude:  float64(51.417331),
		Longitude: float64(-0.284835),
		Timestamp: tempo,
		Active:    true,
		Speed:     float64(0.29 * 1.852001),
		Bearing:   float64(103.93),
	}
	assert.Equal(t, expected, fakeGPSdata)
}

func TestParseCoord(t *testing.T) {
	coord := "5125.0399"
	hemi := "S"
	expected := float64(-51.417331)
	result, err := ParseCoord(coord, hemi)
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
	var fakeGPSdata GPSdata
	err := fakeGPSdata.ParseGPS(input)
	if err == nil {
		t.Fail()
	}
}

func TestParseCoordReturnsError(t *testing.T) {
	wrongCoordInput := "Not a number"
	zero, err := ParseCoord(wrongCoordInput, "E")
	if zero != 0 {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
}
