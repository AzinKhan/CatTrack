package main

import (
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"strconv"
	"strings"
	"time"
)

var UARTPort string

var verbosity int

type GPSdata struct {
	latitude  float64
	longitude float64
	timestamp time.Time
	Active    bool
	speed     float64
	angle     float64
}

const layout string = "020106150405.000"

// GPS gives speed in knots so convert to km/h
const knotRatio float64 = 1.852001

func init() {
	flag.StringVar(&UARTPort, "port", "/dev/ttyS0", "Serial port for connection")
	flag.IntVar(&verbosity, "v", 1, "Verbosity level. Set to zero for silent")
}

func ParseCoord(coord, hemi string) (float64, error) {
	coordinate, err := strconv.ParseFloat(coord, 64)
	if err != nil {
		return 0, err
	}
	switch hemi {
	case "W":
		coordinate = -1.0 * coordinate
	case "S":
		coordinate = -1.0 * coordinate
	}
	return coordinate, nil
}

func (data *GPSdata) ParseGPS(outputline string) error {
	splitz := strings.Split(outputline, ",")
	if len(splitz) != 13 {
		return fmt.Errorf("Not an expected input:\n", splitz)
	}
	var err error
	data.timestamp, err = time.Parse(layout, (splitz[9] + splitz[1]))
	if err != nil {
		return err
	}
	if splitz[2] == "A" {
		data.Active = true
	} else {
		return fmt.Errorf("No fix yet")
	}
	data.latitude, err = ParseCoord(splitz[3], splitz[4])
	if err != nil {
		return err
	}
	data.longitude, err = ParseCoord(splitz[5], splitz[6])
	if err != nil {
		return err
	}
	knotspeed, err := strconv.ParseFloat(splitz[7], 64)
	// Convert to km/h
	data.speed = knotspeed * knotRatio
	if err != nil {
		return err
	}
	data.angle, err = strconv.ParseFloat(splitz[8], 64)
	if err != nil {
		return err
	}
	return nil
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
	var data GPSdata
	for {
		gpsOut := <-gpschan
		if strings.Contains(gpsOut, "GPRMC") {
			err := data.ParseGPS(gpsOut)
			if err != nil {
				log.Println(err)
			}
			if verbosity >= 1 {
				if data.Active {
					log.Println("Latitude:\t", data.latitude)
					log.Println("Longitude:\t", data.longitude)
					log.Println("Time:\t", data.timestamp)
					log.Println("Speed/kmh :\t", data.speed)
					log.Println("Bearing:\t", data.angle, "\n")
				}
			}
		}
	}
}
