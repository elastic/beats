package logstash

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type Config struct {
	Index            string                `config:"index"`
	Port             int                   `config:"port"`
	LoadBalance      bool                  `config:"loadbalance"`
	BulkMaxSize      int                   `config:"bulk_max_size"`
	SlowStart        bool                  `config:"slow_start"`
	Timeout          time.Duration         `config:"timeout"`
	TTL              time.Duration         `config:"ttl"               validate:"min=0"`
	Pipelining       int                   `config:"pipelining"        validate:"min=0"`
	CompressionLevel int                   `config:"compression_level" validate:"min=0, max=9"`
	MaxRetries       int                   `config:"max_retries"       validate:"min=-1"`
	TLS              *outputs.TLSConfig    `config:"ssl"`
	Proxy            transport.ProxyConfig `config:",inline"`
	Backoff          Backoff               `config:"backoff"`
}

type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

var defaultConfig = Config{
	Port:             5044,
	LoadBalance:      false,
	Pipelining:       2,
	BulkMaxSize:      2048,
	SlowStart:        false,
	CompressionLevel: 3,
	Timeout:          30 * time.Second,
	MaxRetries:       3,
	TTL:              0 * time.Second,
	Backoff: Backoff{
		Init: 1 * time.Second,
		Max:  60 * time.Second,
	},
}

func newConfig() *Config {
	c := defaultConfig
	return &c
}
