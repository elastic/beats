package mock

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	pub "github.com/elastic/beats/libbeat/publisher/beat"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct {
	done   chan struct{}
	client pub.Client
}

// Creates beater
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	return &Mockbeat{
		done: make(chan struct{}),
	}, nil
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Run(b *beat.Beat) error {
	var err error

	mb.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	// Wait until mockbeat is done
	mb.client.Publish(pub.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":    "mock",
			"message": "Mockbeat is alive!",
		},
	})
	<-mb.done
	return nil
}

func (mb *Mockbeat) Stop() {
	logp.Info("Mockbeat Stop")

	mb.client.Close()
	close(mb.done)
}
