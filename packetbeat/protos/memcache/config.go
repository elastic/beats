package memcache

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type memcacheConfig struct {
	config.ProtocolCommon `config:",inline"`
	MaxValues             int
	MaxBytesPerValue      int
	UdpTransactionTimeout int
	ParseUnknown          bool
}

var (
	defaultConfig = memcacheConfig{
		ProtocolCommon: config.ProtocolCommon{
			Ports:              []int{11211},
			TransactionTimeout: protos.DefaultTransactionTimeout,
		},
		UdpTransactionTimeout: protos.DefaultTransactionTimeout,
	}
)
