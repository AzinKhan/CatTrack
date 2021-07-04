package gps

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
)

// MockSerialPort mimics the behaviour of a real serial port by
// returning one byte from a given set of bytes (originating as
// a string) every time the Read method is invoked until the bytes
// are exhausted. This implements the io.Reader interface.
type MockSerialPort struct {
	ReturnString string
	ReturnError  error
}

func (m *MockSerialPort) Read(buf []byte) (int, error) {
	returnBytes := []byte(m.ReturnString)
	for i := range buf {
		if len(returnBytes) == 0 {
			return 0, nil
		}
		buf[i] = returnBytes[i]
		m.ReturnString = string(returnBytes[i+1:])
	}
	return len(buf), m.ReturnError
}

func NewMockSerialPort(returnString string, returnError error) *MockSerialPort {
	return &MockSerialPort{
		ReturnString: fmt.Sprintf("$%+v\n", returnString),
		ReturnError:  returnError,
	}
}

func TestReadLine(t *testing.T) {
	testInput := []struct {
		expectedString string
		expectedError  error
	}{
		{"AWholeBunchOfStuff", nil},
		{"", errors.New("TestError")},
	}
	for _, input := range testInput {
		port := NewMockSerialPort(input.expectedString, input.expectedError)
		result, err := readline(port)
		if result != input.expectedString {
			log.Printf("Expected %+v", input.expectedString)
			log.Printf("Got: %+v", result)
			t.Fail()
		}
		if err != input.expectedError {
			log.Printf("Expected %+v", input.expectedError)
			log.Printf("Got: %+v", err)
			t.Fail()
		}
	}
}

func TestReadgps(t *testing.T) {
	messageChan := make(chan string)
	testInput := []struct {
		expectedString string
		expectedError  error
	}{
		{"AWholeBunchOfStuff", nil},
	}
	for _, input := range testInput {
		port := NewMockSerialPort(input.expectedString, input.expectedError)
		go readGPS(context.Background(), port, messageChan)
		result := <-messageChan
		if result != input.expectedString {
			log.Printf("Expected %+v", input.expectedString)
			log.Printf("Got: %+v", result)
			t.Fail()
		}
	}
}
