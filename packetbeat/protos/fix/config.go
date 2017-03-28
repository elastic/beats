package fix

import (
    "github.com/elastic/beats/packetbeat/config"
    "github.com/elastic/beats/packetbeat/protos"
)

type fixConfig struct {
    config.ProtocolCommon `config:",inline"`
}

var (
    defaultConfig = fixConfig{
        ProtocolCommon: config.ProtocolCommon{
            TransactionTimeout: protos.DefaultTransactionExpiration,
        },
    }
)
