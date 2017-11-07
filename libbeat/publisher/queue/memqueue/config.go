package memqueue

import (
	"errors"
	"time"
)

type config struct {
	Events         int           `config:"events" validate:"min=32"`
	FlushMinEvents int           `config:"flush.min_events" validate:"min=0"`
	FlushTimeout   time.Duration `config:"flush.timeout"`
}

var defaultConfig = config{
	Events:         4 * 1024,
	FlushMinEvents: 2 * 1024,
	FlushTimeout:   1 * time.Second,
}

func (c *config) Validate() error {
	if c.FlushMinEvents > c.Events {
		return errors.New("flush.min_events must be less events")
	}

	return nil
}
