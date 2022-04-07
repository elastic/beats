// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/fields"
)

// Config stores the configuration used by the NetFlow Collector.
type Config struct {
	protocols   []string
	logOutput   io.Writer
	expiration  time.Duration
	detectReset bool
	fields      fields.FieldDict
}

var defaultCfg = Config{
	protocols:   []string{},
	logOutput:   ioutil.Discard,
	expiration:  time.Hour,
	detectReset: true,
}

// Defaults returns a configuration object with defaults settings:
// - no protocols are enabled.
// - log output is discarded
// - session expiration is checked once every hour.
func Defaults() Config {
	return defaultCfg
}

// WithProtocols modifies an existing configuration object to enable the
// passed-in protocols.
func (c *Config) WithProtocols(protos ...string) *Config {
	c.protocols = protos
	return c
}

// WithLogOutput sets the output io.Writer for logging.
func (c *Config) WithLogOutput(output io.Writer) *Config {
	c.logOutput = output
	return c
}

// WithExpiration configures the expiration timeout for sessions and templates.
// A value of zero disables expiration.
func (c *Config) WithExpiration(timeout time.Duration) *Config {
	c.expiration = timeout
	return c
}

// WithSequenceResetEnabled allows to toggle the detection of reset sequences,
// which mean that an Exporter has restarted. This will cause the session to be
// reset (all templates expired). A value of true enables this behavior.
func (c *Config) WithSequenceResetEnabled(enabled bool) *Config {
	c.detectReset = enabled
	return c
}

// WithCustomFields extends the NetFlow V9/IPFIX supported fields with
// custom ones. This method can be chained multiple times adding fields
// from different sources.
func (c *Config) WithCustomFields(dicts ...fields.FieldDict) *Config {
	if len(dicts) == 0 {
		return c
	}
	if c.fields == nil {
		c.fields = fields.FieldDict{}
		c.fields.Merge(fields.GlobalFields)
	}
	for _, dict := range dicts {
		c.fields.Merge(dict)
	}
	return c
}

// Protocols returns a list of the protocols enabled.
func (c *Config) Protocols() []string {
	return c.protocols
}

// LogOutput returns the io.Writer where logs are to be written.
func (c *Config) LogOutput() io.Writer {
	return c.logOutput
}

// ExpirationTimeout returns the configured expiration timeout for
// sessions and templates.
func (c *Config) ExpirationTimeout() time.Duration {
	return c.expiration
}

// SequenceResetEnabled returns if sequence reset detection is enabled.
func (c *Config) SequenceResetEnabled() bool {
	return c.detectReset
}

// Fields returns the configured fields.
func (c *Config) Fields() fields.FieldDict {
	if c.fields == nil {
		return fields.GlobalFields
	}
	return c.fields
}
