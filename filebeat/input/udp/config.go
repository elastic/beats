package udp

import (
	"github.com/elastic/beats/filebeat/harvester"
)

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "udp",
	},
	MaxMessageSize: 10240,
	// TODO: What should be default port?
	Host: "localhost:8080",
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Host                      string `config:"host"`
	MaxMessageSize            int    `config:"max_message_size"`
}
