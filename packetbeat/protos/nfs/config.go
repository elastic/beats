package nfs

import (
	"github.com/elastic/beats/packetbeat/config"
	"time"
)

type rpcConfig struct {
	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = rpcConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: 1 * time.Minute,
		},
	}
)
