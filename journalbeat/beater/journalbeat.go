package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/xbeats/journalbeat/config"
	"github.com/elastic/xbeats/journalbeat/reader"
)

type Journalbeat struct {
	done    chan struct{}
	journal *reader.Reader
	config  config.Config
	client  beat.Client
}

func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	r, err := reader.New(config)
	if err != nil {
		return nil, err
	}

	bt := &Journalbeat{
		journal: r,
		done:    make(chan struct{}),
		config:  config,
	}
	return bt, nil
}

func (bt *Journalbeat) Run(b *beat.Beat) error {
	logp.Info("journalbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	for {
		select {
		case <-bt.done:
			return nil
		default:
			for e := range bt.journal.Follow(bt.done) {
				bt.client.Publish(*e)
			}

		}
	}
}

func (bt *Journalbeat) Stop() {
	bt.client.Close()
	bt.journal.Close()
	close(bt.done)
}
