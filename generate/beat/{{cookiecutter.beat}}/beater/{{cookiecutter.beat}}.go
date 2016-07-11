package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"{{cookiecutter.beat_path}}/{{cookiecutter.beat}}/config"
)

type {{cookiecutter.beat|capitalize}} struct {
	config config.Config
	done   chan struct{}
	client publisher.Client
}

// Creates beater
func New() *{{cookiecutter.beat|capitalize}} {
	return &{{cookiecutter.beat|capitalize}}{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (bt *{{cookiecutter.beat|capitalize}}) Config(b *beat.Beat) error {

	bt.config = config.DefaultConfig

	// Load beater config
	err := b.RawConfig.Unpack(&bt.config)
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Setup(b *beat.Beat) error {

	bt.client = b.Publisher.Connect()
	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Run(b *beat.Beat) error {
	logp.Info("{{cookiecutter.beat}} is running! Hit CTRL-C to stop it.")

	ticker := time.NewTicker(bt.config.{{cookiecutter.beat|capitalize}}.Period)
	counter := 1
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       b.Name,
			"counter":    counter,
		}
		bt.client.PublishEvent(event)
		logp.Info("Event sent")
		counter++
	}
}

func (bt *{{cookiecutter.beat|capitalize}}) Cleanup(b *beat.Beat) error {
	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Stop() {
	close(bt.done)
}
