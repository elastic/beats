package httpjsonv2

import (
	"errors"
	"time"
)

type config struct {
	Interval time.Duration  `config:"interval" validate:"required"`
	Auth     *authConfig    `config:"auth"`
	Request  *requestConfig `config:"request" validate:"required"`
	// Response *responseConfig `config:"response"`
}

func (c config) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	return nil
}

func defaultConfig() config {
	return config{
		Interval: time.Minute,
		Auth:     &authConfig{},
		Request: &requestConfig{
			Method: "GET",
		},
	}
}
