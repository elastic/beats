// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-libs/logp"
)

type reloader struct {
	ctx context.Context
	log *logp.Logger
	ch  chan<- []InputConfig

	// validate rejects updates that cannot be applied.
	validate func([]InputConfig) error
}

func (r *reloader) debugLogConfig(cfg *reload.ConfigWithMeta) {
	if !r.log.IsDebug() || cfg == nil || cfg.Config == nil {
		return
	}

	var m map[string]any
	err := cfg.Config.Unpack(&m)
	if err != nil {
		r.log.Debugf("Failed to unpack the config for debug logging: %v", err)
	} else {
		b, _ := json.Marshal(m)
		r.log.Debugf("Reloader config map: %v", string(b))
	}
}

func (r *reloader) Reload(configs []*reload.ConfigWithMeta) error {
	r.log.Debug("Inputs reloader got configuration update")
	inputConfigs := make([]InputConfig, 0)
	for _, cfg := range configs {
		var icfg InputConfig
		err := cfg.Config.Unpack(&icfg)
		if err != nil {
			return r.reject([]*reload.ConfigWithMeta{cfg}, err)
		}

		// Log the new configuration at the debug level only
		r.debugLogConfig(cfg)

		inputConfigs = append(inputConfigs, icfg)
	}

	err := r.validate(inputConfigs)
	if err != nil {
		return r.reject(configs, err)
	}

	select {
	case <-r.ctx.Done():
	default:
		r.ch <- inputConfigs
	}

	return nil
}

// reject logs err and returns it as a unit error for each unit in configs.
// The V2 manager reports only unit errors found in an error list and discards
// any other reload error without a log or status.
func (r *reloader) reject(configs []*reload.ConfigWithMeta, err error) error {
	r.log.Errorf("Rejected osquery input configuration; any configuration already applied stays active and, on the Elastic Agent process runtime, the osquery unit is marked failed: %v", err)
	var errs []error
	seen := make(map[string]bool)
	for _, cfg := range configs {
		if cfg.InputUnitID == "" || seen[cfg.InputUnitID] {
			continue
		}
		seen[cfg.InputUnitID] = true
		errs = append(errs, cfgfile.UnitError{UnitID: cfg.InputUnitID, Err: err})
	}
	if len(errs) == 0 {
		return errors.Join(err)
	}
	return errors.Join(errs...)
}

// WatchInputs returns the input configurations delivered to registry.
// validate is called with each update before it is sent; an update it
// rejects is not sent, and its units are reported as failed.
func WatchInputs(ctx context.Context, log *logp.Logger, registry *reload.Registry, validate func([]InputConfig) error) <-chan []InputConfig {
	ch := make(chan []InputConfig)
	r := &reloader{
		ctx:      ctx,
		log:      log,
		ch:       ch,
		validate: validate,
	}
	registry.MustRegisterInput(r)

	return ch
}
