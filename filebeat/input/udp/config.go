package udp

import (
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/inputsource/udp"
)

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "udp",
	},
	Config: udp.Config{
		MaxMessageSize: 10 * humanize.KiByte,
		// TODO: What should be default port?
		Host: "localhost:8080",
		// TODO: What should be the default timeout?
		Timeout: time.Minute * 5,
	},
}

type config struct {
	udp.Config                `config:",inline"`
	harvester.ForwarderConfig `config:",inline"`
}
