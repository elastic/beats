package kafka

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type kafkaConfig struct {
	Ignore      []string `config:"ignore"`
	SendDetails bool     `config:"send_details"`

	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = kafkaConfig{
		Ignore:      nil,
		SendDetails: true,
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)

func (c *kafkaConfig) Validate() error {
	return nil
}
