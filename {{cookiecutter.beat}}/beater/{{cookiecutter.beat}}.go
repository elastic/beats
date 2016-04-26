package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"{{cookiecutter.beat_path}}/{{cookiecutter.beat}}/config"
)

type {{cookiecutter.beat|capitalize}} struct {
	beatConfig *config.Config
	done       chan struct{}
	period     time.Duration
}

// Creates beater
func New() *{{cookiecutter.beat|capitalize}} {
	return &{{cookiecutter.beat|capitalize}}{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (bt *{{cookiecutter.beat|capitalize}}) Config(b *beat.Beat) error {

	// Load beater beatConfig
	err := b.RawConfig.Unpack(bt.beatConfig)
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Setup(b *beat.Beat) error {

	// Setting default period if not set
	if bt.beatConfig.{{cookiecutter.beat|capitalize}}.Period == "" {
		bt.beatConfig.{{cookiecutter.beat|capitalize}}.Period = "1s"
	}

	var err error
	bt.period, err = time.ParseDuration(bt.beatConfig.{{cookiecutter.beat|capitalize}}.Period)
	if err != nil {
		return err
	}

	return nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Run(b *beat.Beat) error {
	logp.Info("{{cookiecutter.beat}} is running! Hit CTRL-C to stop it.")

	ticker := time.NewTicker(bt.period)
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
		b.Events.PublishEvent(event)
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
