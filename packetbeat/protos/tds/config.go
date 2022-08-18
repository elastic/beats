package tds

import (
	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/elastic-agent-libs/logp"
)

type tdsConfig struct {
	config.ProtocolCommon `config:",inline"`
	MaxRowLength          int `config:"max_row_length"`
	MaxRows               int `config:"max_rows"`
}

var defaultConfig = tdsConfig{
	ProtocolCommon: config.ProtocolCommon{
		TransactionTimeout: protos.DefaultTransactionExpiration,
	},
	MaxRowLength: 1024,
	MaxRows:      10,
}

func (c *tdsConfig) Validate() error {
	logp.Info("config.Validate()")
	return nil
}
