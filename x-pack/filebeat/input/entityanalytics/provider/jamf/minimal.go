// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/entcollect"
	ecjamf "github.com/elastic/entcollect/provider/jamf"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
)

func init() {
	err := provider.RegisterMinimalStateProvider(Name, minimalProvider)
	if err != nil {
		panic(err)
	}
}

func minimalProvider(cfg *config.C, _ *logp.Logger) (entcollect.Provider, time.Duration, time.Duration, error) {
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
	}

	d := ecjamf.DefaultConfig()
	lc := localConf{
		PageSize:       d.PageSize,
		IDSetShards:    d.IDSetShards,
		TokenGrace:     d.TokenGrace,
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
	}
	if err := cfg.Unpack(&lc); err != nil {
		return nil, 0, 0, err
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
		return nil, 0, 0, err
	}
	return ecjamf.New(ec), ec.SyncInterval, ec.UpdateInterval, nil
}
