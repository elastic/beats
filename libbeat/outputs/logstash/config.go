package logstash

import "github.com/elastic/beats/libbeat/outputs"

type logstashConfig struct {
	Index            string             `config:"index"`
	Port             int                `config:"port"`
	LoadBalance      bool               `config:"loadbalance"`
	BulkMaxSize      int                `config:"bulk_max_size"`
	Timeout          int                `config:"timeout"`
	CompressionLevel int                `config:"compression_level"`
	MaxRetries       int                `config:"max_retries"`
	TLS              *outputs.TLSConfig `config:"tls"`
}

var (
	defaultConfig = logstashConfig{
		Port:             10200,
		LoadBalance:      false,
		BulkMaxSize:      2048,
		CompressionLevel: 3,
		Timeout:          30,
		MaxRetries:       3,
		TLS:              nil,
	}
)
