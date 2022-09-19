package decode_duration

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("decode_duration",
		checks.ConfigChecked(NewDecodeDuration,
			checks.RequireFields("field", "format")))
	jsprocessor.RegisterPlugin("DecodeDuration", NewDecodeDuration)
}

type decodeDurationConfig struct {
	Field  string `config:"field"`
	Format string `config:"format"`
}

type decodeDuration struct {
	config decodeDurationConfig
}

func (u decodeDuration) Run(event *beat.Event) (*beat.Event, error) {
	fields := event.Fields
	x, err := fields.GetValue(u.config.Field)
	if err != nil {
		return event, nil
	}
	durationString, ok := x.(string)
	if !ok {
		return event, nil
	}
	d, err := time.ParseDuration(durationString)
	if err != nil {
		return event, nil
	}
	switch u.config.Format {
	case "milliseconds":
		x = d.Seconds() * 1000
	case "seconds":
		x = d.Seconds()
	case "minutes":
		x = d.Minutes()
	case "hours":
		x = d.Hours()
	default:
		x = d.Seconds() * 1000
	}
	_, _ = fields.Put(u.config.Field, x)
	return event, nil
}

func (u decodeDuration) String() string {
	return "decode_duration"
}

func NewDecodeDuration(c *common.Config) (processors.Processor, error) {
	fc := decodeDurationConfig{}
	err := c.Unpack(&fc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack decode duration config: %w", err)
	}

	return &decodeDuration{
		config: fc,
	}, nil
}
