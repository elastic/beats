// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"

	ecjamf "github.com/elastic/entcollect/provider/jamf"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
)

func init() {
	err := provider.RegisterMinimalStateProvider(Name, minimalProvider)
	if err != nil {
		panic(err)
	}
}

func minimalProvider(cfg *config.C, log *logp.Logger) (provider.MinimalState, error) {
	mc, err := minimalConfig(cfg)
	if err != nil {
		return provider.MinimalState{}, err
	}
	client, err := mc.request.Client(log)
	if err != nil {
		return provider.MinimalState{}, fmt.Errorf("creating HTTP client: %w", err)
	}
	return provider.MinimalState{
		Provider:        ecjamf.NewWithClient(mc.provider, client),
		FullSync:        mc.provider.SyncInterval,
		IncrementalSync: mc.provider.UpdateInterval,
		Client:          client,
		Tracer:          mc.tracer,
	}, nil
}

// minimalConf is the minimal-state configuration of the Jamf provider.
type minimalConf struct {
	provider ecjamf.Config
	request  provider.RequestConfig
	tracer   *provider.TracerConfig
}

// minimalConfig unpacks cfg into a validated minimalConf.
func minimalConfig(cfg *config.C) (minimalConf, error) {
	// localConf mirrors ecjamf.Config with config:"" tags for UCF Unpack.
	// If ecjamf.Config gains a new field, add it here and in the mapping
	// below; TestMinimalConfigRoundTrip (minimal_test.go) will catch any drift.
	type localConf struct {
		TenantID       string        `config:"jamf_tenant"        validate:"required"`
		Username       string        `config:"jamf_username"      validate:"required"`
		Password       string        `config:"jamf_password"      validate:"required"`
		PageSize       int           `config:"page_size"`
		IDSetShards    int           `config:"idset_shards"`
		TokenGrace     time.Duration `config:"token_grace_period"`
		SyncInterval   time.Duration `config:"sync_interval"`
		UpdateInterval time.Duration `config:"update_interval"`

		Request provider.RequestConfig `config:"request"`
		Tracer  *provider.TracerConfig `config:"tracer"`
	}

	d := ecjamf.DefaultConfig()
	lc := localConf{
		PageSize:       d.PageSize,
		IDSetShards:    d.IDSetShards,
		TokenGrace:     d.TokenGrace,
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
		Request:        provider.DefaultRequestConfig(30 * time.Second),
	}
	if err := cfg.Unpack(&lc); err != nil {
		return minimalConf{}, err
	}
	ec := ecjamf.Config{
		TenantID:       lc.TenantID,
		Username:       lc.Username,
		Password:       lc.Password,
		PageSize:       lc.PageSize,
		IDSetShards:    lc.IDSetShards,
		TokenGrace:     lc.TokenGrace,
		SyncInterval:   lc.SyncInterval,
		UpdateInterval: lc.UpdateInterval,
	}
	if err := ec.Validate(); err != nil {
		return minimalConf{}, err
	}
	return minimalConf{provider: ec, request: lc.Request, tracer: lc.Tracer}, nil
}
