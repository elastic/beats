package elasticsearch

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

type DeathLetterSelector struct {
	Selector         outputs.IndexSelector
	DeathLetterIndex string
}

func (d DeathLetterSelector) Select(event *beat.Event) (string, error) {
	result, _ := event.Meta.HasKey("deathlettered")
	if result {
		return d.DeathLetterIndex, nil
	}
	return d.Selector.Select(event)
}
