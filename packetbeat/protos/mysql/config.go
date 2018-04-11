package mysql

import (
	"time"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type mysqlConfig struct {
	config.ProtocolCommon `config:",inline"`
	MaxRowLength          int           `config:"max_row_length"`
	MaxRows               int           `config:"max_rows"`
	StatementTimeout      time.Duration `config:"statement_timeout"`
}

var (
	defaultConfig = mysqlConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		MaxRowLength:     1024,
		MaxRows:          10,
		StatementTimeout: 3600 * time.Second,
	}
)
