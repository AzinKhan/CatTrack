package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AzinKhan/CatTrack/gps"
	"github.com/tarm/serial"
)

func main() {
	var UARTPort, addr string
	flag.StringVar(&UARTPort, "port", "/dev/ttyS0", "Serial port for connection")
	flag.StringVar(&addr, "s", "http://localhost:8000", "Address of remote server")
	flag.Parse()

	config := &serial.Config{
		Name: UARTPort,
		Baud: 9600,
	}
	log.Println("Opening port", UARTPort)
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-ch
		log.Printf("Signal %s received, exiting...\n", s)
		cancel()
	}()

	reader := gps.NewReader(port)

	for message := range reader.ReadGPS(ctx) {
		if !strings.Contains(message, "GPRMC") {
			continue
		}
		log.Printf("Read GPS output: %s", message)

		urlData := url.Values{}
		urlData.Add("Output", message)
		resp, err := client.PostForm(addr+"/marker", urlData)
		if err != nil {
			log.Println(err)
			continue
		}
		resp.Body.Close()
	}
}
