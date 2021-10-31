// Copyright 2018 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package discovery

import (
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/keys"

	"github.com/open-policy-agent/opa/bundle"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/download"
	"github.com/open-policy-agent/opa/util"
)

// Config represents the configuration for the discovery feature.
type Config struct {
	download.Config                            // bundle downloader configuration
	Name            *string                    `json:"name"`               // name of the discovery bundle
	Prefix          *string                    `json:"prefix,omitempty"`   // Deprecated: use `Resource` instead.
	Decision        *string                    `json:"decision"`           // the name of the query to run on the bundle to get the config
	Service         string                     `json:"service"`            // the name of the service used to download discovery bundle from
	Resource        *string                    `json:"resource,omitempty"` // the resource path which will be downloaded from the service
	Signing         *bundle.VerificationConfig `json:"signing,omitempty"`  // configuration used to verify a signed bundle

	service string
	path    string
	query   string
}

// ConfigBuilder assists in the construction of the plugin configuration.
type ConfigBuilder struct {
	raw      []byte
	services []string
	keys     map[string]*keys.Config
}

// NewConfigBuilder returns a new ConfigBuilder to build and parse the discovery config
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{}
}

// WithBytes sets the raw discovery config
func (b *ConfigBuilder) WithBytes(config []byte) *ConfigBuilder {
	b.raw = config
	return b
}

// WithServices sets the services that implement control plane APIs
func (b *ConfigBuilder) WithServices(services []string) *ConfigBuilder {
	b.services = services
	return b
}

// WithKeyConfigs sets the public keys to verify a signed bundle
func (b *ConfigBuilder) WithKeyConfigs(keys map[string]*keys.Config) *ConfigBuilder {
	b.keys = keys
	return b
}

// Parse returns a valid Config object with defaults injected.
func (b *ConfigBuilder) Parse() (*Config, error) {
	if b.raw == nil {
		return nil, nil
	}

	var result Config

	if err := util.Unmarshal(b.raw, &result); err != nil {
		return nil, err
	}

	return &result, result.validateAndInjectDefaults(b.services, b.keys)
}

// ParseConfig returns a valid Config object with defaults injected.
func ParseConfig(bs []byte, services []string) (*Config, error) {
	return NewConfigBuilder().WithBytes(bs).WithServices(services).Parse()
}

func (c *Config) validateAndInjectDefaults(services []string, confKeys map[string]*keys.Config) error {

	if c.Resource == nil && c.Name == nil {
		return fmt.Errorf("missing required discovery.resource field")
	}

	// make a copy of the keys map
	copy := map[string]*keys.Config{}
	for key, kc := range confKeys {
		copy[key] = kc
	}

	if c.Signing != nil {
		err := c.Signing.ValidateAndInjectDefaults(copy)
		if err != nil {
			return fmt.Errorf("invalid configuration for discovery service %q: %s", *c.Name, err.Error())
		}
	} else {
		if len(confKeys) > 0 {
			c.Signing = bundle.NewVerificationConfig(copy, "", "", nil)
		}
	}

	if c.Resource != nil {
		c.path = *c.Resource
	} else {
		if c.Prefix == nil {
			s := defaultDiscoveryPathPrefix
			c.Prefix = &s
		}

		c.path = fmt.Sprintf("%v/%v", strings.Trim(*c.Prefix, "/"), strings.Trim(*c.Name, "/"))
	}

	service, err := c.getServiceFromList(c.Service, services)
	if err != nil {
		return fmt.Errorf("invalid configuration for discovery service: %s", err.Error())
	}

	c.service = service

	if c.Decision != nil {
		c.query = fmt.Sprintf("%v.%v", ast.DefaultRootDocument, strings.Replace(strings.Trim(*c.Decision, "/"), "/", ".", -1))
	} else if c.Name != nil {
		c.query = fmt.Sprintf("%v.%v", ast.DefaultRootDocument, strings.Replace(strings.Trim(*c.Name, "/"), "/", ".", -1))
	} else {
		c.query = ast.DefaultRootDocument.String()
	}

	return c.Config.ValidateAndInjectDefaults()
}

func (c *Config) getServiceFromList(service string, services []string) (string, error) {
	if service == "" {
		if len(services) != 1 {
			return "", fmt.Errorf("more than one service is defined")
		}
		return services[0], nil
	}
	for _, svc := range services {
		if svc == service {
			return service, nil
		}
	}
	return service, fmt.Errorf("service name %q not found", service)
}

const (
	defaultDiscoveryPathPrefix = "bundles"
)
