package mock

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct {
	done   chan struct{}
	client publisher.Client
}

// Creates beater
func New() *Mockbeat {
	return &Mockbeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Config(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Setup(b *beat.Beat) error {
	mb.client = b.Publisher.Connect()
	return nil
}

func (mb *Mockbeat) Run(b *beat.Beat) error {
	// Wait until mockbeat is done
	mb.client.PublishEvent(common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "mock",
		"message":    "Mockbeat is alive!",
	})
	<-mb.done
	return nil
}

func (mb *Mockbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (mb *Mockbeat) Stop() {
	close(mb.done)
	mb.client.Close()
}
