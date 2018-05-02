package syslog

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/filebeat/inputsource/tcp"
	"github.com/elastic/beats/filebeat/inputsource/udp"
	"github.com/elastic/beats/libbeat/common"
)

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Protocol                  common.ConfigNamespace `config:"protocol"`
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "syslog",
	},
}

var defaultTCP = tcp.Config{
	LineDelimiter:  "\n",
	Timeout:        time.Minute * 5,
	MaxMessageSize: 20 * humanize.MiByte,
}

var defaultUDP = udp.Config{
	MaxMessageSize: 10 * humanize.KiByte,
	Timeout:        time.Minute * 5,
}

func factory(
	cb inputsource.NetworkFunc,
	config common.ConfigNamespace,
) (inputsource.Network, error) {
	n, cfg := config.Name(), config.Config()

	switch n {
	case tcp.Name:
		config := defaultTCP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
		return tcp.New(&config, cb)
	case udp.Name:
		config := defaultUDP
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
		return udp.New(&config, cb), nil
	default:
		return nil, fmt.Errorf("you must choose between TCP or UDP")
	}
}
