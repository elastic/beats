// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/common/reload"
)

type reloader struct {
	ctx context.Context
	ch  chan<- []InputConfig
}

func (r *reloader) Reload(configs []*reload.ConfigWithMeta) error {
	var inputConfigs []InputConfig
	for _, cfg := range configs {
		var icfg InputConfig
		err := cfg.Config.Unpack(&icfg)
		if err != nil {
			return err
		}
		inputConfigs = append(inputConfigs, icfg)
	}

	select {
	case <-r.ctx.Done():
	default:
		r.ch <- inputConfigs
	}

	return nil
}

func WatchInputs(ctx context.Context) <-chan []InputConfig {
	ch := make(chan []InputConfig)
	r := &reloader{
		ctx: ctx,
		ch:  ch,
	}
	reload.Register.MustRegisterList("inputs", r)

	return ch
}
