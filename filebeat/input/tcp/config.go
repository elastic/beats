package tcp

import (
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/inputsource/tcp"
)

type config struct {
	tcp.Config                `config:",inline"`
	harvester.ForwarderConfig `config:",inline"`
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "tcp",
	},
	Config: tcp.Config{
		LineDelimiter:  "\n",
		Timeout:        time.Minute * 5,
		MaxMessageSize: 20 * humanize.MiByte,
	},
}
