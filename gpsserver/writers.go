package gpsserver

import (
	"context"
	"log"
)

// Logger is an implementation of the DataWriter
// interface which simply logs the data received.
type Logger struct{}

func (l *Logger) Write(ctx context.Context, data GPSReading) error {
	log.Printf("Received data %+v", data)
	return nil
}
