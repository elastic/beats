package logstash

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type logstashConfig struct {
	Index            string                `config:"index"`
	Port             int                   `config:"port"`
	LoadBalance      bool                  `config:"loadbalance"`
	BulkMaxSize      int                   `config:"bulk_max_size"`
	Timeout          time.Duration         `config:"timeout"`
	Pipelining       int                   `config:"pipelining"        validate:"min=0"`
	CompressionLevel int                   `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries       int                   `config:"max_retries"       validate:"min=-1"`
	TLS              *outputs.TLSConfig    `config:"ssl"`
	Proxy            transport.ProxyConfig `config:",inline"`
}

var (
	defaultConfig = logstashConfig{
		Port:             10200,
		LoadBalance:      false,
		BulkMaxSize:      2048,
		CompressionLevel: 3,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
	}
)
