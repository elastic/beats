package http

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type httpConfig struct {
	config.ProtocolCommon `config:",inline"`
	Send_all_headers      bool     `config:"send_all_headers"`
	Send_headers          []string `config:"send_headers"`
	Split_cookie          bool     `config:"split_cookie"`
	Real_ip_header        string   `config:"real_ip_header"`
	Include_body_for      []string `config:"include_body_for"`
	Hide_keywords         []string `config:"hide_keywords"`
	Redact_authorization  bool     `config:"redact_authorization"`
}

var (
	defaultConfig = httpConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)
