package gps

import (
	"context"
	"io"
	"log"
)

const (
	startCharacter = 0x0024
	newline        = 0x000a
	carriageReturn = 0x000d
)

type Reader struct {
	io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r}
}

func (r *Reader) ReadGPS(ctx context.Context) chan string {
	ch := make(chan string)
	go readGPS(ctx, r, ch)
	return ch
}

// readGPS reads from a given io.Reader and calls Readline. The message is then
// written to a given channel.
func readGPS(ctx context.Context, r io.Reader, ch chan string) {
	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		default:
			message, err := readline(r)
			if err != nil {
				log.Printf("Error reading line: %+v", err)
				continue
			}
			ch <- message
		}
	}
}

// readline reads from r and waits for a starting character, $ at which point
// it will read from the port until a newline or return character is reached.
// It returns a string of the read line.
func readline(r io.Reader) (string, error) {
	line := make([]byte, 0, 255)
	buf := make([]byte, 1)
	// Wait for the start character
	for {
		n, err := r.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		if buf[0] == startCharacter {
			break
		}
	}
mainloop:
	for {
		n, err := r.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		switch buf[0] {
		case carriageReturn, newline:
			break mainloop
		}
		line = append(line, buf[0])
	}
	return string(line), nil
}
