// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/confmap"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"
)

// Config is config settings for metricbeat receiver.  The structure of
// which is the same as the metricbeat.yml configuration file.
type Config struct {
	Beatconfig map[string]any `mapstructure:",remain"`
}

// Unmarshal implements confmap.Unmarshaler for custom unmarshaling logic.
func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := xpInstance.ConvertPaths(conf); err != nil {
		return fmt.Errorf("error converting paths: %w", err)
	}
	if err := conf.Unmarshal(c); err != nil {
		return fmt.Errorf("error unmarshalling conf: %w", err)
	}
	return nil
}

// Validate checks if the configuration in valid
func (c *Config) Validate() error {
	if len(c.Beatconfig) == 0 {
		return fmt.Errorf("configuration is required")
	}

	if _, prs := c.Beatconfig["metricbeat"]; !prs {
		return fmt.Errorf("configuration key 'metricbeat' is required")
	}
	return nil
}
