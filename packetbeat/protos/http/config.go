package http

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

type httpConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendAllHeaders        bool     `config:"send_all_headers"`
	SendHeaders           []string `config:"send_headers"`
	SplitCookie           bool     `config:"split_cookie"`
	RealIPHeader          string   `config:"real_ip_header"`
	IncludeBodyFor        []string `config:"include_body_for"`
	HideKeywords          []string `config:"hide_keywords"`
	RedactAuthorization   bool     `config:"redact_authorization"`
	MaxMessageSize        int      `config:"max_message_size"`
}

var (
	defaultConfig = httpConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		MaxMessageSize: tcp.TCPMaxDataInStream,
	}
)
