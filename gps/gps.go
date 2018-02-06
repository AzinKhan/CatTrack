package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/tarm/serial"
)

var UARTPort, serverIP string

func init() {
	flag.StringVar(&UARTPort, "port", "/dev/ttyS0", "Serial port for connection")
	flag.StringVar(&serverIP, "s", "localhost:8000", "Address of remote server")
}

// Readline takes a serial port and waits for a starting character, $
// at which point it will read from the port until a newline or
// return character is reached. It returns a string of the read line.
func Readline(port *serial.Port) string {
	line := make([]byte, 0, 255)
	buf := make([]byte, 1)
	// Wait for the start character - $
firstLoop:
	for {
		n, _ := port.Read(buf)
		if n > 0 {
			if string(buf[0]) == "$" {
				break firstLoop
			}
		}
	}
mainloop:
	for {
		n, _ := port.Read(buf)
		if n > 0 {
			onebuf := string(buf[0])
			if onebuf == "\n" || onebuf == "\r" {
				break mainloop
			}
			line = append(line, buf[0])
		}
	}
	return string(line)
}

// Readgps opens a port on the given device name and reads from it
// using Readline. The message is then written to a given channel.
func Readgps(portname string, ch chan string) {
	config := &serial.Config{
		Name: portname,
		Baud: 9600,
	}
	log.Println("Opening port", portname)
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}
	for {
		message := Readline(port)
		ch <- message
	}
}

func main() {
	flag.Parse()
	gpschan := make(chan string)
	go Readgps(UARTPort, gpschan)
	for {
		gpsOut := <-gpschan
		if strings.Contains(gpsOut, "GPRMC") {
			urlData := url.Values{}
			urlData.Add("Output", gpsOut)
			http.PostForm(serverIP+"/marker", urlData)
		}
	}
}
