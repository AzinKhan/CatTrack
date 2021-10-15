package gpsserver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	receiverBuffer  = 10
	publisherBuffer = 100
	receiveTimeout  = 5 * time.Second
)

// DataWriter is an interface for different clients that may
// receive the GPS data. Implementations may write the data,
// for example, to a database, a websocket, or stdout.
type DataWriter interface {
	Write(context.Context, *GPSReading) error
}

type Publisher struct {
	ch        chan *GPSReading
	receivers map[string]chan *GPSReading
	mu        *sync.Mutex
}

func NewPublisher() *Publisher {
	return &Publisher{
		ch:        make(chan *GPSReading, publisherBuffer),
		receivers: make(map[string]chan *GPSReading),
		mu:        new(sync.Mutex),
	}
}

func NewRunningPublisher(ctx context.Context) *Publisher {
	p := NewPublisher()
	go p.Run(ctx)
	return p
}

func (p *Publisher) Run(ctx context.Context) {
	log.Println("Running publisher")
	for {
		select {
		case data := <-p.ch:
			p.mu.Lock()
			log.Printf("Sending data to %d receivers", len(p.receivers))
			var wg sync.WaitGroup
			wg.Add(len(p.receivers))
			for id, receiver := range p.receivers {
				rcvr := receiver
				rcvrID := id
				// Send data in a goroutine with a timeout to ensure that
				// one blocking receiver (due to full channel) does not
				// block the rest.
				go func() {
					defer wg.Done()
					sendWithTimeout(rcvrID, data, rcvr)
				}()
			}
			wg.Wait()
			p.mu.Unlock()

		case <-ctx.Done():
			log.Printf("Context cancelled, stopping Publisher")
			return
		}
	}
}

func sendWithTimeout(id string, data *GPSReading, receiver chan (*GPSReading)) {
	done := make(chan struct{})
	timer := time.NewTimer(receiveTimeout)
	// TODO: If this times out then the goroutine will leak
	go func() {
		receiver <- data
		timer.Stop()
		done <- struct{}{}
	}()

	select {
	case <-timer.C:
		log.Printf("Receiver %s timed out", id)
	case <-done:
	}

}

// AddReceiver adds a receiver to the Publisher's subscribers and returns a function
// for removing that subscriber. The receiver will execute the given DataWriter's
// Write function when it receives data.
func (p *Publisher) AddReceiver(ctx context.Context, dw DataWriter) (remove func()) {
	log.Printf("Adding receiver %+v", dw)
	childCtx, cancel := context.WithCancel(ctx)
	receiveCh := make(chan *GPSReading, receiverBuffer)

	rcvr := &receiver{receiveCh, dw}
	id := uuid.New().String()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.receivers[id] = receiveCh

	go rcvr.run(childCtx)

	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		log.Println("CANCELLING")
		cancel()
		delete(p.receivers, id)
	}
}

// Publish sends the given data point to the connected subscribers.
// TODO: Data should not be a pointer
func (p *Publisher) Publish(ctx context.Context, data *GPSReading) {
	if len(p.receivers) == 0 {
		log.Println("No receivers, skipping publish")
		return
	}
	p.ch <- data
}

type receiver struct {
	ch <-chan *GPSReading
	dw DataWriter
}

func (r *receiver) run(ctx context.Context) {
	log.Println("Running receiver")
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping receiver")
			return
		case data := <-r.ch:
			err := r.dw.Write(ctx, data)
			if err != nil {
				log.Printf("Error writing data: %v\n", err)
			}
		}
	}
}
