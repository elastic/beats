package mongodb

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type mongodbConfig struct {
	config.ProtocolCommon `config:",inline"`
	MaxDocLength          int `config:"max_doc_length"`
	MaxDocs               int `config:"max_docs"`
}

var (
	defaultConfig = mongodbConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		MaxDocLength: 5000,
		MaxDocs:      10,
	}
)
