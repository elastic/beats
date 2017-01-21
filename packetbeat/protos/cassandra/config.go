package cassandra

import (
	"fmt"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"

	gocql "github.com/elastic/beats/packetbeat/protos/cassandra/internal/gocql"
)

type cassandraConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendRequestHeader     bool            `config:"send_request_header"`
	SendResponseHeader    bool            `config:"send_response_header"`
	Compressor            string          `config:"compressor"`
	OPsIgnored            []gocql.FrameOp `config:"ignored_ops"`
}

var (
	defaultConfig = cassandraConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
			SendRequest:        true,
			SendResponse:       true,
		},
		SendRequestHeader:  true,
		SendResponseHeader: true,
	}
)

func (c *cassandraConfig) Validate() error {
	if !(c.Compressor == "" || c.Compressor == "snappy") {
		return fmt.Errorf("invalid compressor config: %s, only snappy supported", c.Compressor)
	}
	return nil
}
