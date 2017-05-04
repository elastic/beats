// Author: Sudhindra Sajjal <sudhindra.s@media.net> & Sumit Sulakhe <sumit.su@media.net>
// Copyright: Media.net 2017

package graphite

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type graphiteConfig struct {
	config.ProtocolCommon `config:",inline"`
}

var (
	defaultConfig = graphiteConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)

func (c *graphiteConfig) Validate() error {
	return nil
}
