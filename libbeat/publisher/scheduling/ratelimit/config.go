package ratelimit

import "errors"

type config struct {
	EventsPerSecond uint `config:"events_per_second" validate:"min=1"`
	Shared          bool `config:"shared"`
}

func defaultConfig() config {
	return config{
		Shared: true,
	}
}

func (c *config) Validate() error {
	if c.EventsPerSecond == 0 {
		return errors.New("events_per_second must be > 0")
	}

	return nil
}
