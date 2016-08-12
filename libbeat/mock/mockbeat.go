package mock

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

type Mockbeat struct{}

// Creates beater
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	return &Mockbeat{}, nil
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Run(b *beat.Beat) error {
	client := b.Publisher.Connect()
	b.Done.OnStop.Close(client)

	// Wait until mockbeat is done
	client.PublishEvent(common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "mock",
		"message":    "Mockbeat is alive!",
	})

	b.Done.Wait()
	return nil
}
