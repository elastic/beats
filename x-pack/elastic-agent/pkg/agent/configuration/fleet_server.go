// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configuration

import (
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// FleetServerConfig is the configuration written so Elastic Agent can run Fleet Server.
type FleetServerConfig struct {
	Bootstrap bool                     `config:"bootstrap" yaml:"bootstrap,omitempty"`
	Policy    *FleetServerPolicyConfig `config:"policy" yaml:"policy,omitempty"`
	Output    FleetServerOutputConfig  `config:"output" yaml:"output,omitempty"`
}

// FleetServerPolicyConfig is the configuration for the policy Fleet Server should run on.
type FleetServerPolicyConfig struct {
	ID string `config:"id"`
}

// FleetServerOutputConfig is the connection for Fleet Server to call to Elasticsearch.
type FleetServerOutputConfig struct {
	Elasticsearch Elasticsearch `config:"elasticsearch" yaml:"elasticsearch"`
}

// Elasticsearch is the configuration for elasticsearch.
type Elasticsearch struct {
	Protocol string            `config:"protocol" yaml:"protocol"`
	Hosts    []string          `config:"hosts" yaml:"hosts"`
	Path     string            `config:"path" yaml:"path,omitempty"`
	Username string            `config:"username" yaml:"username"`
	Password string            `config:"password" yaml:"password"`
	TLS      *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty"`
}

// ElasticsearchFromConnStr returns an Elasticsearch configuration from the connection string.
func ElasticsearchFromConnStr(conn string) (Elasticsearch, error) {
	u, err := url.Parse(conn)
	if err != nil {
		return Elasticsearch{}, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Elasticsearch{}, errors.New("invalid connection string: scheme must be http or https")
	}
	if u.Host == "" {
		return Elasticsearch{}, errors.New("invalid connection string: must include a host")
	}
	if u.User == nil || u.User.Username() == "" {
		return Elasticsearch{}, errors.New("invalid connection string: must include a username")
	}
	password, ok := u.User.Password()
	if !ok {
		return Elasticsearch{}, errors.New("invalid connection string: must include a password")
	}
	return Elasticsearch{
		Protocol: u.Scheme,
		Hosts:    []string{u.Host},
		Path:     u.Path,
		Username: u.User.Username(),
		Password: password,
		TLS:      nil,
	}, nil
}
