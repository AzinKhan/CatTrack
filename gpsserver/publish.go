package gpsserver

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
)

const (
	receiverBuffer  = 10
	publisherBuffer = 100
)

// DataWriter is an interface for different clients that may
// receive the GPS data. Implementations may write the data,
// for example, to a database, a websocket, or stdout.
type DataWriter interface {
	Write(context.Context, *GPSdata) error
}

type Publisher struct {
	ch        chan *GPSdata
	receivers map[string]chan *GPSdata
	mu        *sync.Mutex
}

func NewPublisher() *Publisher {
	return &Publisher{
		ch:        make(chan *GPSdata, publisherBuffer),
		receivers: make(map[string]chan *GPSdata),
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
			for _, receiver := range p.receivers {
				receiver <- data
			}
			p.mu.Unlock()

		case <-ctx.Done():
			log.Printf("Context cancelled, stopping Publisher")
			return
		}
	}
}

// AddReceiver adds a receiver to the Publisher's subscribers and returns a function
// for removing that subscriber. The receiver will execute the given DataWriter's
// Write function when it receives data.
func (p *Publisher) AddReceiver(ctx context.Context, dw DataWriter) (remove func()) {
	log.Printf("Adding receiver %+v", dw)
	childCtx, cancel := context.WithCancel(ctx)
	receiveCh := make(chan *GPSdata, receiverBuffer)

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
func (p *Publisher) Publish(ctx context.Context, data *GPSdata) {
	if len(p.receivers) == 0 {
		log.Println("No receivers, skipping publish")
		return
	}
	p.ch <- data
}

type receiver struct {
	ch <-chan *GPSdata
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
