// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"time"
)

type Config struct {
	Paths []string `config:"paths"`

	Backoff       time.Duration `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor int           `config:"backoff_factor" validate:"min=1"`
	MaxBackoff    time.Duration `config:"max_backoff" validate:"min=0,nonzero"`
}

var DefaultConfig = Config{
	Paths: make([]string, 0),

	Backoff:       1 * time.Second,
	BackoffFactor: 2,
	MaxBackoff:    30 * time.Second,
}
