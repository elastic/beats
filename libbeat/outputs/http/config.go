package http

import (
	"time"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type httpConfig struct {
	Protocol         string            `config:"protocol"`
	Path             string            `config:"path"`
	Params           map[string]string `config:"parameters"`
	Username         string            `config:"username"`
	Password         string            `config:"password"`
	ProxyURL         string            `config:"proxy_url"`
	LoadBalance      bool              `config:"loadbalance"`
	BatchPublish     bool              `config:"batch_publish"`
	BatchSize        int               `config:"batch_size"`
	CompressionLevel int               `config:"compression_level" validate:"min=0, max=9"`
	TLS              *tlscommon.Config `config:"tls"`
	MaxRetries       int               `config:"max_retries"`
	Timeout          time.Duration     `config:"timeout"`
	Headers          map[string]string `config:"headers"`
	ContentType      string            `config:"content_type"`
	Backoff          backoff           `config:"backoff"`
	Format           string            `config:"format"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

var (
	defaultConfig = httpConfig{
		Protocol:         "",
		Path:             "",
		Params:           nil,
		ProxyURL:         "",
		Username:         "",
		Password:         "",
		BatchPublish:     false,
		BatchSize:        2048,
		Timeout:          90 * time.Second,
		CompressionLevel: 0,
		TLS:              nil,
		MaxRetries:       3,
		LoadBalance:      false,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		Format:           "json",
	}
)

func (c *httpConfig) Validate() error {
	if c.ProxyURL != "" {
		if _, err := parseProxyURL(c.ProxyURL); err != nil {
			return err
		}
	}
	if c.Format != "json" && c.Format != "json_lines" {
		return fmt.Errorf("Unsupported config option format: %s", c.Format)
	}

	return nil
}
