package mock

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	return &Mockbeat{
		done: make(chan struct{}),
	}, nil
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Run(b *beat.Beat) error {
	mb.client = b.Publisher.Connect()

	// Wait until mockbeat is done
	mb.client.PublishEvent(common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "mock",
		"message":    "Mockbeat is alive!",
	})
	<-mb.done
	return nil
}

func (mb *Mockbeat) Stop() {
	logp.Info("Mockbeat Stop")

	mb.client.Close()
	close(mb.done)
}
