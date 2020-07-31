// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Config is the configuration for the Kibana client.
type Config struct {
	Protocol Protocol          `config:"protocol" yaml:"protocol"`
	SpaceID  string            `config:"space.id" yaml:"space.id,omitempty"`
	Username string            `config:"username" yaml:"username,omitempty"`
	Password string            `config:"password" yaml:"password,omitempty"`
	Path     string            `config:"path" yaml:"path,omitempty"`
	Host     string            `config:"host" yaml:"host,omitempty"`
	Hosts    []string          `config:"hosts" yaml:"hosts,omitempty"`
	Timeout  time.Duration     `config:"timeout" yaml:"timeout,omitempty"`
	TLS      *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty"`
}

// Protocol define the protocol to use to make the connection. (Either HTTPS or HTTP)
type Protocol string

const (
	// ProtocolHTTP is HTTP protocol connection to Kibana.
	ProtocolHTTP Protocol = "http"
	// ProtocolHTTPS is HTTPS protocol connection to Kibana.
	ProtocolHTTPS Protocol = "https"
)

// Unpack the protocol.
func (p *Protocol) Unpack(from string) error {
	if Protocol(from) != ProtocolHTTPS && Protocol(from) != ProtocolHTTP {
		return fmt.Errorf("invalid protocol %s, accepted values are 'http' and 'https'", from)
	}

	*p = Protocol(from)
	return nil
}

// DefaultClientConfig creates default configuration for kibana client.
func DefaultClientConfig() *Config {
	return &Config{
		Protocol: ProtocolHTTP,
		Host:     "localhost:5601",
		Path:     "",
		SpaceID:  "",
		Username: "",
		Password: "",
		Timeout:  90 * time.Second,
		TLS:      nil,
	}
}

// IsBasicAuth returns true if the username and password are both defined.
func (c *Config) IsBasicAuth() bool {
	return len(c.Username) > 0 && len(c.Password) > 0
}

// GetHosts returns the hosts to connect to kibana.
//
// This looks first at `Hosts` and then at `Host` when `Hosts` is not defined.
func (c *Config) GetHosts() []string {
	if len(c.Hosts) > 0 {
		return c.Hosts
	}
	return []string{c.Host}
}
