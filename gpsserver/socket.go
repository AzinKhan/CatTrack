package gpsserver

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type SocketWriter struct {
	conn *websocket.Conn
	ch   chan *GPSReading
}

// NewSocketWriter returns a SocketWriter that will write
// any GPSData received via its Write method to the given
// websocket connection.
func NewSocketWriter(conn *websocket.Conn) *SocketWriter {
	return &SocketWriter{
		conn: conn,
		ch:   make(chan *GPSReading, 100),
	}
}

func (s *SocketWriter) Write(ctx context.Context, data *GPSReading) error {
	s.ch <- data
	return nil
}

func (s *SocketWriter) Run(ctx context.Context) {
	log.Println("Running socketwriter")
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled for socket writer")
			s.conn.Close()
			return
		case data := <-s.ch:
			b, err := json.Marshal(data)
			if err != nil {
				log.Println(err)
				continue
			}
			err = s.conn.WriteMessage(1, b)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}
