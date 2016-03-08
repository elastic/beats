package logstash

import (
	"fmt"
	"net/url"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"

	"golang.org/x/net/proxy"
)

type logstashConfig struct {
	Index            string             `config:"index"`
	Port             int                `config:"port"`
	LoadBalance      bool               `config:"loadbalance"`
	BulkMaxSize      int                `config:"bulk_max_size"`
	Timeout          int                `config:"timeout"`
	CompressionLevel int                `config:"compression_level"`
	MaxRetries       int                `config:"max_retries"`
	TLS              *outputs.TLSConfig `config:"tls"`
	Proxy            proxyConfig        `config:",inline"`
}

var (
	defaultConfig = logstashConfig{
		Port:             10200,
		LoadBalance:      false,
		BulkMaxSize:      2048,
		CompressionLevel: 3,
		Timeout:          30,
		MaxRetries:       3,
	}
)

// proxyConfig holds the configuration information required to proxy
// Logstash connections through a SOCKS5 proxy server.
type proxyConfig struct {
	URL          string `config:"proxy_url"`                // URL of the SOCKS proxy. Scheme must be socks5. Username and password can be embedded in the URL.
	LocalResolve bool   `config:"proxy_use_local_resolver"` // Resolve names locally instead of on the SOCKS server.

	parsedURL *url.URL // Parsed copy of URL.
}

// parseURL parses the socks5 proxy URL and verifies that it is a well-formed.
// If the URL is not set then this becomes a no-op. If the URL is not a valid
// socks5 URL then an error will be returned.
func (s *proxyConfig) parseURL() error {
	if s.URL == "" {
		return nil
	}

	var err error
	if s.parsedURL, err = url.Parse(s.URL); err != nil {
		return fmt.Errorf("proxy_url: %v", err)
	}

	_, err = proxy.FromURL(s.parsedURL, nil)
	if err == nil {
		logp.Info("SOCKS5 proxy host: '%s'", s.parsedURL.Host)
	}
	return err
}
