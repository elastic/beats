package elasticsearch

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
)

type elasticsearchConfig struct {
	Protocol         string             `config:"protocol"`
	Path             string             `config:"path"`
	Params           map[string]string  `config:"parameters"`
	Headers          map[string]string  `config:"headers"`
	Username         string             `config:"username"`
	Password         string             `config:"password"`
	ProxyURL         string             `config:"proxy_url"`
	LoadBalance      bool               `config:"loadbalance"`
	CompressionLevel int                `config:"compression_level" validate:"min=0, max=9"`
	TLS              *outputs.TLSConfig `config:"ssl"`
	BulkMaxSize      int                `config:"bulk_max_size"`
	MaxRetries       int                `config:"max_retries"`
	Timeout          time.Duration      `config:"timeout"`
	Backoff          Backoff            `config:"backoff"`
}

type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

const (
	defaultBulkSize = 50
)

var (
	defaultConfig = elasticsearchConfig{
		Protocol:         "",
		Path:             "",
		ProxyURL:         "",
		Params:           nil,
		Username:         "",
		Password:         "",
		Timeout:          90 * time.Second,
		MaxRetries:       3,
		CompressionLevel: 0,
		TLS:              nil,
		LoadBalance:      true,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
)

func (c *elasticsearchConfig) Validate() error {
	if c.ProxyURL != "" {
		if _, err := parseProxyURL(c.ProxyURL); err != nil {
			return err
		}
	}

	return nil
}
