package gpsserver

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func MockServerFactory() (string, *http.ServeMux, func()) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	return srv.URL, mux, srv.Close
}

type dataTester struct {
	t        *testing.T
	expected *GPSdata
}

func (dt *dataTester) Write(ctx context.Context, data *GPSdata) error {
	assert.Equal(dt.t, dt.expected, data)
	return nil
}

func TestLocationHandler(t *testing.T) {
	serverURL, mux, tearDown := MockServerFactory()
	defer tearDown()
	ctx := context.Background()
	publisher := NewRunningPublisher(ctx)
	mux.HandleFunc("/marker", NewLocationHandler(publisher))
	tempo, _ := time.Parse(layout, "030218224537.000")
	testInput := []struct {
		testLine       string
		expectedText   string
		expectedStatus int
		expectedData   *GPSdata
	}{
		{
			"GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79",
			"Location updated",
			http.StatusOK,
			&GPSdata{
				Latitude:  float64(51.417331),
				Longitude: float64(-0.284835),
				Timestamp: tempo,
				Active:    true,
				Speed:     float64(0.29 * 1.852001),
				Bearing:   float64(103.93),
			},
		},
		{
			"NOTVALID", "Error parsing gps output: Could not parse timestamp\n",
			http.StatusBadRequest,
			nil,
		},
	}
	for _, input := range testInput {
		urlData := url.Values{}
		urlData.Add("Output", input.testLine)
		remove := publisher.AddReceiver(ctx, &dataTester{t, input.expectedData})
		response, err := http.PostForm(serverURL+"/marker", urlData)

		assert.NoError(t, err)
		assert.Equal(t, input.expectedStatus, response.StatusCode)
		responsetext, err := ioutil.ReadAll(response.Body)
		assert.NoError(t, err)
		assert.Equal(t, input.expectedText, string(responsetext))
		remove()
	}
}

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
