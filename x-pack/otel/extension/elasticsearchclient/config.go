// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"fmt"
	"sort"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	defaultRequestTimeout  = 10 * time.Second
	defaultIdleConnTimeout = 3 * time.Second
)

// Config contains the Elasticsearch endpoint and HTTP transport settings.
type Config struct {
	Hosts      []string          `config:"hosts"`
	Protocol   string            `config:"protocol"`
	Path       string            `config:"path"`
	Parameters map[string]string `config:"parameters"`
	Headers    map[string]string `config:"headers"`

	Username string `config:"username"`
	Password string `config:"password"`
	APIKey   string `config:"api_key"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

var _ confmap.Unmarshaler = (*Config)(nil)

func createDefaultConfig() component.Config {
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = defaultRequestTimeout
	transport.IdleConnTimeout = defaultIdleConnTimeout
	return &Config{Transport: transport}
}

// Unmarshal decodes the Collector configuration through the Elastic transport
// decoder. The custom boundary is needed because the common transport types
// use Elastic config tags and custom Unpack methods rather than mapstructure.
func (c *Config) Unmarshal(component *confmap.Conf) error {
	if component == nil || len(component.ToStringMap()) == 0 {
		return nil
	}

	values := component.ToStringMap()
	if err := rejectUnknownKeys(values); err != nil {
		return err
	}

	input, err := config.NewConfigFrom(values)
	if err != nil {
		return fmt.Errorf("invalid elasticsearchclient configuration: %w", err)
	}

	decoded := struct {
		Hosts      []string          `config:"hosts"`
		Protocol   string            `config:"protocol"`
		Path       string            `config:"path"`
		Parameters map[string]string `config:"parameters"`
		Headers    map[string]string `config:"headers"`

		Username string `config:"username"`
		Password string `config:"password"`
		APIKey   string `config:"api_key"`

		Transport httpcommon.HTTPTransportSettings `config:",inline"`
	}{Transport: c.Transport}

	if err := input.Unpack(&decoded); err != nil {
		return fmt.Errorf("invalid elasticsearchclient configuration: %w", err)
	}

	*c = Config{
		Hosts:      decoded.Hosts,
		Protocol:   decoded.Protocol,
		Path:       decoded.Path,
		Parameters: decoded.Parameters,
		Headers:    decoded.Headers,
		Username:   decoded.Username,
		Password:   decoded.Password,
		APIKey:     decoded.APIKey,
		Transport:  decoded.Transport,
	}
	return nil
}

var acceptedKeys = map[string]struct{}{
	"api_key":                 {},
	"headers":                 {},
	"hosts":                   {},
	"idle_connection_timeout": {},
	"parameters":              {},
	"password":                {},
	"path":                    {},
	"protocol":                {},
	"proxy_disable":           {},
	"proxy_headers":           {},
	"proxy_url":               {},
	"ssl":                     {},
	"timeout":                 {},
	"username":                {},
}

var acceptedTLSKeys = map[string]struct{}{
	"ca_sha256":                  {},
	"ca_trusted_fingerprint":     {},
	"certificate":                {},
	"certificate_authorities":    {},
	"certificate_reload":         {},
	"cipher_suites":              {},
	"curve_types":                {},
	"disable_legacy_pem_support": {},
	"enabled":                    {},
	"key":                        {},
	"key_passphrase":             {},
	"key_passphrase_path":        {},
	"renegotiation":              {},
	"supported_protocols":        {},
	"verification_mode":          {},
}

var acceptedCertificateReloadKeys = map[string]struct{}{
	"enabled":         {},
	"reload_interval": {},
}

func rejectUnknownKeys(values map[string]any) error {
	var unknown []string
	for key := range values {
		if _, ok := acceptedKeys[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if ssl, ok := values["ssl"]; ok && ssl != nil {
		sslValues, ok := ssl.(map[string]any)
		if !ok {
			return fmt.Errorf("ssl must be a map")
		}
		for key := range sslValues {
			if _, ok := acceptedTLSKeys[key]; !ok {
				unknown = append(unknown, "ssl."+key)
			}
		}
		if reload, ok := sslValues["certificate_reload"]; ok && reload != nil {
			reloadValues, ok := reload.(map[string]any)
			if !ok {
				return fmt.Errorf("ssl.certificate_reload must be a map")
			}
			for key := range reloadValues {
				if _, ok := acceptedCertificateReloadKeys[key]; !ok {
					unknown = append(unknown, "ssl.certificate_reload."+key)
				}
			}
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("unsupported elasticsearchclient configuration key(s): %s", unknown)
}

// Validate checks that the extension has a usable Elasticsearch endpoint and
// internally consistent transport settings.
func (c *Config) Validate() error {
	if len(c.Hosts) == 0 {
		return fmt.Errorf("elasticsearchclient requires at least one host")
	}
	if c.APIKey != "" && (c.Username != "" || c.Password != "") {
		return fmt.Errorf("cannot set both api_key and username/password")
	}
	if c.Transport.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	if c.Transport.IdleConnTimeout < 0 {
		return fmt.Errorf("idle_connection_timeout must not be negative")
	}
	if proxy := c.Transport.Proxy.URL; proxy != nil {
		_, err := httpcommon.NewHTTPClientProxySettings(
			proxy.String(),
			map[string]string(c.Transport.Proxy.Headers),
			c.Transport.Proxy.Disable,
		)
		if err != nil {
			return fmt.Errorf("invalid proxy configuration: %w", err)
		}
	}
	if c.Transport.TLS != nil {
		if err := c.Transport.TLS.Validate(); err != nil {
			return fmt.Errorf("invalid TLS configuration: %w", err)
		}
		if err := c.Transport.TLS.VerificationMode.Validate(); err != nil {
			return fmt.Errorf("invalid TLS configuration: %w", err)
		}
	}
	return nil
}
