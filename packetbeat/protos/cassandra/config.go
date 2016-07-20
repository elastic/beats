package cassandra

import (
	"fmt"
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/pkg/errors"
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
	if !(c.Compressor == "" || c.Compressor == "snappy") {
		return errors.New(fmt.Sprintf("invalid compressor config: %s, only snappy supported", c.Compressor))
	}
	return nil
}
