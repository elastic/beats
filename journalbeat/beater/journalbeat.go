package beater

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/journalbeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/journalbeat/config"
)

type Journalbeat struct {
	input  *input.Input
	done   chan struct{}
	config config.Config

	client   beat.Client
	pipeline beat.Pipeline
}

func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	client, err := b.Publisher.Connect()
	if err != nil {
		return err
	}

	done := make(chan struct{})
	i, err := input.New(config, client, done)
	if err != nil {
		return nil, err
	}

	bt := &Journalbeat{
		input:  i,
		done:   done,
		config: config,
		client: client,
	}
	return bt, nil
}

func (bt *Journalbeat) Run(b *beat.Beat) error {
	logp.Info("journalbeat is running! Hit CTRL-C to stop it.")
	defer logp.Info("journalbeat is stopping")

	var wg sync.WaitGroup
	wg.Add(1)
	go runInput(&wg)
	wg.Wait()

	return nil
}

func (bt *Journalbeat) runInput(wg *sync.WaitGroup) {
	defer wg.Done()
	bt.input.Run()
}

func (bt *Journalbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
