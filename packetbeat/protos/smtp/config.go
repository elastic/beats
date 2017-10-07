package smtp

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type smtpConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendDataHeaders       bool `config:"send_data_headers"`
	SendDataBody          bool `config:"send_data_body"`
}

var (
	defaultConfig = smtpConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
			SendRequest:        true,
			SendResponse:       true,
		},
	}
)

func (c *smtpConfig) Validate() error {
	return nil
}
