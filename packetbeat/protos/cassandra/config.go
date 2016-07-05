package cassandra

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type cassandraConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendRequestHeader     bool   `config:"send_request_header"`
	SendResponseHeader    bool   `config:"send_response_header"`
	Compressor            string `config:"compressor"`
}

var (
	defaultConfig = cassandraConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		SendRequestHeader:  false,
		SendResponseHeader: false,
	}
)

func (c *cassandraConfig) Validate() error {
	return nil
}
