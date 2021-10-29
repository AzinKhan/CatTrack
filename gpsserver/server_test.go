package gpsserver

import (
	"context"
	"io/ioutil"
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
	expected GPSReading
}

func (dt *dataTester) Write(ctx context.Context, data GPSReading) error {
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
		expectedData   GPSReading
	}{
		{
			"GPRMC,224537.000,A,5125.0399,N,00017.0901,W,0.29,103.93,030218,,,D*79",
			"Location updated",
			http.StatusOK,
			GPSReading{
				Latitude:  float64(51.417331),
				Longitude: float64(-0.284835),
				Timestamp: tempo,
				Active:    true,
				Speed:     float64(0.29 * 1.852001),
				Bearing:   float64(103.93),
			},
		},
		{
			// TODO: Don't check error strings
			"NOTVALID", "Error parsing gps output: could not parse timestamp\n",
			http.StatusBadRequest,
			GPSReading{},
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
