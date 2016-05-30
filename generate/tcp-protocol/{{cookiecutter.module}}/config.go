package {{ cookiecutter.module }}

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type {{ cookiecutter.module }}Config struct {
	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = {{ cookiecutter.module }}Config{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)

func (c *{{ cookiecutter.module }}Config) Validate() error {
	return nil
}
