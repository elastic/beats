package nfs

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type rpcConfig struct {
	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = rpcConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)
