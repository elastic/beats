package dns

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type dnsConfig struct {
	config.ProtocolCommon `config:",inline"`
	Include_authorities   bool `config:"include_authorities"`
	Include_additionals   bool `config:"include_additionals"`
}

var (
	defaultConfig = dnsConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)
