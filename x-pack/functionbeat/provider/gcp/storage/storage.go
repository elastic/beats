// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"context"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/functionbeat/config"
	"github.com/elastic/beats/x-pack/functionbeat/function/beater"
	"github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp"
	"github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp/transformer"
	_ "github.com/elastic/beats/x-pack/functionbeat/provider/gcp/include"
)

func RunCloudStorage(ctx context.Context, e transformer.StorageEvent) error {
	settings := instance.Settings{
		Name:            "functionbeat",
		IndexPrefix:     "functionbeat",
		ConfigOverrides: config.FunctionOverrides,
	}

	cfgfile.ChangeDefaultCfgfileFlag(settings.Name)

	return instance.Run(settings, initFunctionbeat(ctx, e))
}

func initFunctionbeat(ctx context.Context, e transformer.StorageEvent) func(*beat.Beat, *common.Config) (beat.Beater, error) {
	return func(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
		bt, err := beater.New(b, cfg)
		if err != nil {
			return nil, err
		}

		fnbeat, ok := bt.(*beater.Functionbeat)
		if !ok {
			return nil, fmt.Errorf("not Functionbeat")
		}

		fnbeat.Ctx = context.WithValue(fnbeat.Ctx, gcp.StorageContext(gcp.StorageContextCtxStr), ctx)
		fnbeat.Ctx = context.WithValue(fnbeat.Ctx, gcp.StorageMsg(gcp.StorageEvtCtxStr), e)

		return fnbeat, nil
	}
}
