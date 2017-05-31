package tds

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type tdsConfig struct {
	config.ProtocolCommon `config: ",inline"`
	// MaxRowLength          int `config: "max_row_length"`
	// MaxRows               int `config: "max_rows"`
}

var (
	defaultConfig = tdsConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		// MaxRowLength: 1024,
		// MaxRows:      10,
	}
)
