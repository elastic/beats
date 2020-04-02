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
	Timeout  time.Duration     `config:"timeout" yaml:"timeout,omitempty"`
	TLS      *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty"`
}

// Protocol define the protocol to use to make the connection. (Either HTTPS or HTTP)
type Protocol string

// Unpack the protocol.
func (p *Protocol) Unpack(from string) error {
	if from != "https" && from != "http" {
		return fmt.Errorf("invalid protocol %s, accepted values are 'http' and 'https'", from)
	}
	return nil
}

func defaultClientConfig() Config {
	return Config{
		Protocol: Protocol("http"),
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
