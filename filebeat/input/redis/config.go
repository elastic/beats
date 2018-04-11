package redis

import (
	"time"

	"github.com/elastic/beats/filebeat/harvester"
)

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "redis",
	},
	Network:  "tcp",
	MaxConn:  10,
	Password: "",
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Hosts                     []string      `config:"hosts" validate:"required"`
	IdleTimeout               time.Duration `config:"idle_timeout"`
	Network                   string        `config:"network"`
	MaxConn                   int           `config:"maxconn" validate:"min=1"`
	Password                  string        `config:"password"`
}
