package gpsserver

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func MockServerFactory() (string, *http.ServeMux, func()) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	return srv.URL, mux, srv.Close
}

func TestUpdateMarkerGet(t *testing.T) {
	serverURL, mux, tearDown := MockServerFactory()
	defer tearDown()
	tempo, _ := time.Parse(layout, "030218224537.000")
	fakeGPSdata := GPSdata{
		Latitude:  float64(51.417331),
		Longitude: float64(-0.284835),
		Timestamp: tempo,
		Active:    true,
		Speed:     float64(0.29 * 1.852001),
		Bearing:   float64(103.93),
	}
	mux.HandleFunc("/marker", UpdateMarker(&fakeGPSdata))
	response, err := http.Get(serverURL + "/marker")
	if err != nil {
		t.Fail()
	}
	if response.StatusCode != 200 {
		t.Fail()
	}
	resp, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fail()
	}
	var resultData GPSdata
	err = json.Unmarshal(resp, &resultData)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	if resultData != fakeGPSdata {
		t.Fail()
	}
}

func TestUpdateMarkerPost(t *testing.T) {
	serverURL, mux, tearDown := MockServerFactory()
	defer tearDown()
	var fakeGPSdata GPSdata
	mux.HandleFunc("/marker", UpdateMarker(&fakeGPSdata))
	testInput := []struct {
		testLine       string
		expectedText   string
		expectedStatus int
		//expectedTime   time.Time
	}{
		{
			"GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79",
			"Location updated",
			200,
		},
		{
			"NOTVALID", "Error parsing gps output: not an expected input [NOTVALID]",
			400,
		},
	}
	for _, input := range testInput {
		urlData := url.Values{}
		urlData.Add("Output", input.testLine)
		response, err := http.PostForm(serverURL+"/marker", urlData)
		if err != nil {
			t.Fail()
		}
		if response.StatusCode != input.expectedStatus {
			t.Fail()
		}
		responsetext, _ := ioutil.ReadAll(response.Body)
		if string(responsetext) != input.expectedText {
			log.Printf("Got response text: %+v", string(responsetext))
			log.Printf("Expected response text: %+v", input.expectedText)
			t.Fail()
		}
		if input.expectedStatus == 200 {
			tempo, _ := time.Parse(layout, "030218224537.000")
			log.Println(tempo)
			expected := GPSdata{
				Latitude:  float64(51.417331),
				Longitude: float64(-0.284835),
				Timestamp: tempo,
				Active:    true,
				Speed:     float64(0.29 * 1.852001),
				Bearing:   float64(103.93),
			}
			if fakeGPSdata != expected {
				t.Fail()
			}
		}
	}
}

func TestUpdateMarkerUnsupportedMethod(t *testing.T) {
	serverURL, mux, tearDown := MockServerFactory()
	defer tearDown()
	var fakeGPSdata GPSdata
	mux.HandleFunc("/marker", UpdateMarker(&fakeGPSdata))

	response, err := http.Head(serverURL + "/marker")

	if err != nil {
		t.Fail()
	}
	expectedStatus := 400
	if response.StatusCode != expectedStatus {
		log.Printf("Received status %+v", response.StatusCode)
		log.Printf("Expected status code %+v", expectedStatus)
		t.Fail()
	}
	log.Printf("%+v", response)

}

func TestParseGPS(t *testing.T) {
	var fakeGPSdata GPSdata
	testLine := "GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79"
	fakeGPSdata.ParseGPS(testLine)
	tempo, _ := time.Parse(layout, "030218224537.000")
	expected := GPSdata{
		Latitude:  float64(51.417331),
		Longitude: float64(-0.284835),
		Timestamp: tempo,
		Active:    true,
		Speed:     float64(0.29 * 1.852001),
		Bearing:   float64(103.93),
	}
	if fakeGPSdata != expected {
		t.Fail()
	}
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
