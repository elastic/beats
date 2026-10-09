// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"fmt"

	xpInstance "github.com/elastic/beats/v7/x-pack/libbeat/cmd/instance"
	conf "github.com/elastic/elastic-agent-libs/config"

	"go.opentelemetry.io/collector/confmap"
)

// Config is config settings for heartbeat receiver.  The structure of
// which is the same as the heartbeat.yml configuration file, except for the
// receiver specific settings declared as fields below.
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

	Beatconfig map[string]any `mapstructure:",remain"`
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

	hb, prs := c.Beatconfig["heartbeat"]
	if !prs {
		return fmt.Errorf("configuration key 'heartbeat' is required")
	}

	// A run_once scheduler waits for every job added to it, which is not
	// meaningful for receivers that share one, or that a collector restarts.
	if hbMap, ok := hb.(map[string]any); ok {
		runOnce, err := conf.NewConfigFrom(hbMap)
		if err != nil {
			return fmt.Errorf("error reading 'heartbeat' configuration: %w", err)
		}
		var parsed struct {
			RunOnce bool `config:"run_once"`
		}
		if err := runOnce.Unpack(&parsed); err != nil {
			return fmt.Errorf("error reading 'heartbeat.run_once': %w", err)
		}
		if parsed.RunOnce {
			return fmt.Errorf("'heartbeat.run_once' is not supported by the heartbeat receiver")
		}
	}
	return nil
}
