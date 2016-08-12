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
	config     config.Config
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &{{cookiecutter.beat|capitalize}}{
		config: config,
	}
	return bt, nil
}

func (bt *{{cookiecutter.beat|capitalize}}) Run(b *beat.Beat) error {
	logp.Info("{{cookiecutter.beat}} is running! Hit CTRL-C to stop it.")

	client = b.Publisher.Connect()
	b.Done.OnStop.Close(client)

	ticker := time.NewTicker(bt.config.Period)
	counter := 1
	for {
		select {
		case <-b.Done.C:
			return nil
		case <-ticker.C:
		}

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       b.Name,
			"counter":    counter,
		}
		client.PublishEvent(event)
		logp.Info("Event sent")
		counter++
	}
}
