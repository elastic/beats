package dhcp

import (
	"github.com/elastic/beats/packetbeat/config"
)

type dhcpConfig struct {
	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = dhcpConfig{}
)
