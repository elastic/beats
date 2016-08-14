package lumberjack

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type lumberjackConfig struct {
	config.ProtocolCommon `config:",inline"`
	OutOfBandData         bool `config:"out_of_band_data"`
}

var (
	defaultConfig = lumberjackConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)

func (c *lumberjackConfig) Validate() error {
	return nil
}
