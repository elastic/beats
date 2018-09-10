// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type Config struct {
	Inputs        []*common.Config  `config:"inputs"`
	RegistryFile  string            `config:"registry"`
	Backoff       time.Duration     `config:"backoff" validate:"min=0,nonzero"`
	BackoffFactor int               `config:"backoff_factor" validate:"min=1"`
	MaxBackoff    time.Duration     `config:"max_backoff" validate:"min=0,nonzero"`
	Matches       map[string]string `config:"matches"`
	Seek          string            `config:"seek"`
}

var DefaultConfig = Config{
	RegistryFile:  "registry",
	Backoff:       1 * time.Second,
	BackoffFactor: 2,
	MaxBackoff:    30 * time.Second,
	Seek:          "cursor",
}
