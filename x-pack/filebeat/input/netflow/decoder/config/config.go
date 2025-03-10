// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/elastic-agent-libs/logp"
)

type ActiveSessionsMetric interface {
	Inc()
	Dec()
}

// Config stores the configuration used by the NetFlow Collector.
type Config struct {
	protocols            []string
	logOutput            *logp.Logger
	expiration           time.Duration
	detectReset          bool
	fields               fields.FieldDict
	sharedTemplates      bool
	withCache            bool
	activeSessionsMetric ActiveSessionsMetric
}

// Defaults returns a configuration object with defaults settings:
// - no protocols are enabled.
// - log output is set to the logger that is passed in.
// - session expiration is checked once every hour.
// - resets are detected.
// - templates are not shared.
// - cache is disabled.
func Defaults(logger *logp.Logger) Config {
	return Config{
		protocols:       []string{},
		logOutput:       logger,
		expiration:      time.Hour,
		detectReset:     true,
		sharedTemplates: false,
		withCache:       false,
	}
}

// WithProtocols modifies an existing configuration object to enable the
// passed-in protocols.
func (c *Config) WithProtocols(protos ...string) *Config {
	c.protocols = protos
	return c
}

// WithExpiration configures the expiration timeout for sessions and templates.
// A value of zero disables expiration.
func (c *Config) WithExpiration(timeout time.Duration) *Config {
	c.expiration = timeout
	return c
}

// WithCache toggles the packet cache.
func (c *Config) WithCache(enabled bool) *Config {
	c.withCache = enabled
	return c
}

// Cache returns if the packet cache is enabled.
func (c *Config) Cache() bool {
	return c.withCache
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

// WithSharedTemplates allows to toggle the sharing of templates within
// a v9 neflow or ipfix session. If it is not enabled, the source address
// must match the address of the source of the template.
func (c *Config) WithSharedTemplates(enabled bool) *Config {
	c.sharedTemplates = enabled
	return c
}

// WithActiveSessionsMetric configures the metric used to report active sessions.
func (c *Config) WithActiveSessionsMetric(metric ActiveSessionsMetric) *Config {
	c.activeSessionsMetric = metric
	return c
}

// Protocols returns a list of the protocols enabled.
func (c *Config) Protocols() []string {
	return c.protocols
}

// LogOutput returns the io.Writer where logs are to be written.
func (c *Config) LogOutput() *logp.Logger {
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

// ShareTemplatesEnabled returns if template sharing is enabled.
func (c *Config) ShareTemplatesEnabled() bool {
	return c.sharedTemplates
}

// Fields returns the configured fields.
func (c *Config) Fields() fields.FieldDict {
	if c.fields == nil {
		return fields.GlobalFields
	}
	return c.fields
}

// ActiveSessionsMetric returns the configured metric to track active sessions.
func (c *Config) ActiveSessionsMetric() ActiveSessionsMetric {
	if c == nil {
		return nil
	}

	return c.activeSessionsMetric
}
