package gps

import (
	"io"
	"log"

	"github.com/tarm/serial"
)

// Readline takes a serial port and waits for a starting character, $
// at which point it will read from the port until a newline or
// return character is reached. It returns a string of the read line.
func Readline(port io.Reader) (string, error) {
	line := make([]byte, 0, 255)
	buf := make([]byte, 1)
	// Wait for the start character - $
firstLoop:
	for {
		n, err := port.Read(buf)
		if err != nil {
			return "", err
		}
		if n > 0 {
			if string(buf[0]) == "$" {
				break firstLoop
			}
		}
	}
mainloop:
	for {
		n, err := port.Read(buf)
		if err != nil {
			return "", err
		}
		if n > 0 {
			onebuf := string(buf[0])
			if onebuf == "\n" || onebuf == "\r" {
				break mainloop
			}
			line = append(line, buf[0])
		}
	}
	return string(line), nil
}

// Readgps reads from a given io.Reader (assumed to be a serial.Port)
// and calls Readline. The message is then written to a given channel.
func Readgps(port io.Reader, ch chan string) {
	for {
		message, err := Readline(port)
		if err != nil {
			log.Printf("Error reading line: %+v", err)
			continue
		}
		ch <- message
	}
}

func InitializePort(portname string) (*serial.Port, error) {
	config := &serial.Config{
		Name: portname,
		Baud: 9600,
	}
	log.Println("Opening port", portname)
	return serial.OpenPort(config)
}
