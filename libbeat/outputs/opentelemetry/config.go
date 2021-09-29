package opentelemetry

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type otelConfig struct {
	UserAgent    string                `config:"user_agent"`
	Endpoint       string                `config:"endpoint"`
	Key         string                `config:"key"`
	LoadBalance bool                  `config:"loadbalance"`
	Timeout     time.Duration         `config:"timeout"`
	BulkMaxSize int                   `config:"bulk_max_size"`
	MaxRetries  int                   `config:"max_retries"`
	TLS         *tlscommon.Config     `config:"ssl"`
	Proxy       transport.ProxyConfig `config:",inline"`
	DataSource    string                `config:"datasource"`
	Backoff     backoff               `config:"backoff"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

var (
	defaultConfig = otelConfig{
		LoadBalance: true,
		Timeout:     5 * time.Second,
		BulkMaxSize: 2048,
		MaxRetries:  3,
		TLS:         nil,
		DataSource:    "metrics",
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
)

func (c *otelConfig) Validate() error {
	return nil
}
