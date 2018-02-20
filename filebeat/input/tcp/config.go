package tcp

import (
	"time"

	"github.com/elastic/beats/filebeat/harvester"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Host                      string        `config:"host"`
	LineDelimiter             string        `config:"line_delimiter" validate:"nonzero"`
	Timeout                   time.Duration `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize            uint64        `config:"max_message_size" validate:"nonzero,positive"`
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "tcp",
	},
	LineDelimiter:  "\n",
	Host:           "localhost:9000",
	Timeout:        time.Minute * 5,
	MaxMessageSize: 20 * 1024 * 1024,
}
