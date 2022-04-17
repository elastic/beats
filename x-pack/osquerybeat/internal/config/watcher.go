// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"context"
	"encoding/json"

	"github.com/menderesk/beats/v7/libbeat/common/reload"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

type reloader struct {
	ctx context.Context
	log *logp.Logger
	ch  chan<- []InputConfig
}

func (r *reloader) debugLogConfig(cfg *reload.ConfigWithMeta) {
	if !r.log.IsDebug() || cfg == nil || cfg.Config == nil {
		return
	}

	var m map[string]interface{}
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
	var inputConfigs []InputConfig
	for _, cfg := range configs {
		var icfg InputConfig
		err := cfg.Config.Unpack(&icfg)
		if err != nil {
			return err
		}

		// Log the new configuration at the debug level only
		r.debugLogConfig(cfg)

		inputConfigs = append(inputConfigs, icfg)
	}

	select {
	case <-r.ctx.Done():
	default:
		r.ch <- inputConfigs
	}

	return nil
}

func WatchInputs(ctx context.Context, log *logp.Logger) <-chan []InputConfig {
	ch := make(chan []InputConfig)
	r := &reloader{
		ctx: ctx,
		log: log,
		ch:  ch,
	}
	reload.Register.MustRegisterList("inputs", r)

	return ch
}
