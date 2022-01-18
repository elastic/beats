// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
)

// Config is the configuration for the client.
type Config struct {
	Protocol Protocol `config:"protocol" yaml:"protocol"`
	SpaceID  string   `config:"space.id" yaml:"space.id,omitempty"`
	Path     string   `config:"path" yaml:"path,omitempty"`
	Host     string   `config:"host" yaml:"host,omitempty"`
	Hosts    []string `config:"hosts" yaml:"hosts,omitempty"`

	Transport httpcommon.HTTPTransportSettings `config:",inline" yaml:",inline"`
}

// Protocol define the protocol to use to make the connection. (Either HTTPS or HTTP)
type Protocol string

const (
	// ProtocolHTTP is HTTP protocol connection.
	ProtocolHTTP Protocol = "http"
	// ProtocolHTTPS is HTTPS protocol connection.
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

// DefaultClientConfig creates default configuration for client.
func DefaultClientConfig() Config {
	transport := httpcommon.DefaultHTTPTransportSettings()
	// Default timeout 10 minutes, expecting Fleet Server to control the long poll with default timeout of 5 minutes
	transport.Timeout = 10 * time.Minute

	return Config{
		Protocol:  ProtocolHTTP,
		Host:      "localhost:5601",
		Path:      "",
		SpaceID:   "",
		Transport: transport,
	}
}

// GetHosts returns the hosts to connect.
//
// This looks first at `Hosts` and then at `Host` when `Hosts` is not defined.
func (c *Config) GetHosts() []string {
	if len(c.Hosts) > 0 {
		return c.Hosts
	}
	return []string{c.Host}
}
