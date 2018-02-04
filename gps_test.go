package main

import (
	"testing"
	"time"
)

func TestParseGPS(t *testing.T) {
	var fakeGPSdata GPSdata
	testLine := "GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79"
	fakeGPSdata.ParseGPS(testLine)
	if !fakeGPSdata.Active {
		t.Fail()
	}
	expectedSpeed := float64(0.29 * 1.852001)
	if fakeGPSdata.speed != expectedSpeed {
		t.Fail()
	}
	expectedAngle := float64(103.93)
	if fakeGPSdata.angle != expectedAngle {
		t.Fail()
	}
	expectedTime, _ := time.Parse(layout, "030218224537.000")
	if fakeGPSdata.timestamp != expectedTime {
		t.Fail()
	}
	expectedLatitude := float64(5125.0399)
	if fakeGPSdata.latitude != expectedLatitude {
		t.Fail()
	}
	expectedLongitude := float64(-17.0901)
	if fakeGPSdata.longitude != expectedLongitude {
		t.Fail()
	}
}

func TestParseCoord(t *testing.T) {
	coord := "17.0901"
	hemi := "W"
	expected := float64(-17.0901)
	result, err := ParseCoord(coord, hemi)
	if err != nil {
		t.Fail()
	}
	if result != expected {
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
