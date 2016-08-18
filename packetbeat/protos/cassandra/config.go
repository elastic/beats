package cassandra

import (
	"fmt"
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
	. "github.com/elastic/beats/packetbeat/protos/cassandra/internal/gocql"
	"github.com/pkg/errors"
)

type cassandraConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendRequestHeader     bool      `config:"send_request_header"`
	SendResponseHeader    bool      `config:"send_response_header"`
	Compressor            string    `config:"compressor"`
	OPsIgnored            []FrameOp `config:"ignored_ops"`
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
		return errors.New(fmt.Sprintf("invalid compressor config: %s, only snappy supported", c.Compressor))
	}
	return nil
}
