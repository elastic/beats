package main

import "github.com/elastic/beats/filebeat/harvester"

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Protocol                  string    `config:"protocol"`
	UDP                       udpConfig `config:"udp"`
}

type udpConfig struct {
	Host           string `config:"host"`
	MaxMessageSize int    `config:"max_message_size"`
}

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "syslog",
	},
	Protocol: "UDP",
	Udp:      defaultUDPConfig,
}

var defaultUDPConfig = udpConfig{
	MaxMessageSize: 10240,
	Host:           "localhost:8080",
}

func (c *config) isUDP() bool {
	return strings.Upper(c.Protocol) == "UDP"
}
