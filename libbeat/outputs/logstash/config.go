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
	CompressionLevel int                   `config:"compression_level"`
	MaxRetries       int                   `config:"max_retries"`
	TLS              *outputs.TLSConfig    `config:"tls"`
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
