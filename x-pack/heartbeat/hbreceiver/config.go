// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"

	"go.opentelemetry.io/collector/confmap"
)

<<<<<<< HEAD
// Config is config settings for heartbeat receiver.  The structure of
// which is the same as the heartbeat.yml configuration file.
type Config struct {
	Beatconfig map[string]any `mapstructure:",remain"`
=======
// Config contains receiver-only settings and the remaining Heartbeat
// configuration.
type Config struct {
	// SchedulerGroup makes this receiver share its scheduler, and therefore its
	// `heartbeat.scheduler.limit` and `heartbeat.jobs.<type>.limit` concurrency
	// bounds, with every other heartbeat receiver in this collector configured
	// with the same group. A Heartbeat process only ever has one scheduler, so
	// grouping the receivers that were split out of one Heartbeat process keeps
	// those limits meaningful.
	//
	// The first receiver of a group to start configures its scheduler, later
	// receivers with conflicting limits log a warning.
	//
	// Defaults to the empty group, which every receiver without an explicit
	// group shares.
	SchedulerGroup string `mapstructure:"scheduler_group"`

	MonitorState MonitorStateConfig `mapstructure:"monitor_state"`
	Beatconfig   map[string]any     `mapstructure:",remain"`
}

// MonitorStateConfig configures monitor state restoration.
type MonitorStateConfig struct {
	Elasticsearch string `mapstructure:"elasticsearch"`
>>>>>>> 188f7ba ([Heartbeat] Prevent Synthetics monitor state from resetting after Elastic Agent restarts (#53183))
}

// Unmarshal implements confmap.Unmarshaler for custom unmarshaling logic.
func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := xpInstance.DeDotKeys(conf); err != nil {
		return fmt.Errorf("error converting paths: %w", err)
	}

	// Deep-merge factory defaults into the user-supplied conf so that
	// partial overrides (e.g. only path.home) preserve sibling defaults
	// (e.g. path.data). We merge defaults first, then re-apply the
	// original user values on top so user settings always win.
	if len(c.Beatconfig) > 0 {
		userMap := conf.ToStringMap()
		if err := conf.Merge(confmap.NewFromStringMap(c.Beatconfig)); err != nil {
			return fmt.Errorf("error merging defaults: %w", err)
		}
		if err := conf.Merge(confmap.NewFromStringMap(userMap)); err != nil {
			return fmt.Errorf("error re-applying user config: %w", err)
		}
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

	if _, prs := c.Beatconfig["heartbeat"]; !prs {
		return fmt.Errorf("configuration key 'heartbeat' is required")
	}
	return nil
}
