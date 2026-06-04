// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/entcollect"
	ecentraid "github.com/elastic/entcollect/provider/entraid"
)

func init() {
	err := provider.RegisterMinimalStateProvider(Name, minimalProvider)
	if err != nil {
		panic(err)
	}
}

func minimalProvider(cfg *config.C, _ *logp.Logger) (entcollect.Provider, time.Duration, time.Duration, error) {
	// localConf mirrors ecentraid.Config with config:"" tags for ucfg
	// Unpack. The config tag names match the legacy azuread provider's
	// config namespace so that existing beats YAML works unchanged.
	// If ecentraid.Config gains a new field, add it here and in the
	// mapping below; TestMinimalConfigRoundTrip will catch any drift.
	type selection struct {
		Users   []string `config:"users"`
		Groups  []string `config:"groups"`
		Devices []string `config:"devices"`
	}
	type localConf struct {
		TenantID string `config:"tenant_id" validate:"required"`
		ClientID string `config:"client_id" validate:"required"`
		Secret   string `config:"secret"    validate:"required"`

		LoginEndpoint string   `config:"login_endpoint"`
		LoginScopes   []string `config:"login_scopes"`
		APIEndpoint   string   `config:"api_endpoint"`

		Dataset    string   `config:"dataset"`
		EnrichWith []string `config:"enrich_with"`

		Select selection `config:"select"`

		SyncInterval   time.Duration `config:"sync_interval"`
		UpdateInterval time.Duration `config:"update_interval"`
	}

	d := ecentraid.DefaultConfig()
	lc := localConf{
		LoginEndpoint: d.LoginEndpoint,
		LoginScopes:   d.LoginScopes,
		APIEndpoint:   d.APIEndpoint,
		Dataset:       d.Dataset,
		EnrichWith:    d.EnrichWith,
		Select: selection{
			Users:   d.SelectUsers,
			Groups:  d.SelectGroups,
			Devices: d.SelectDevices,
		},
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
	}
	if err := cfg.Unpack(&lc); err != nil {
		return nil, 0, 0, err
	}

	ec := ecentraid.Config{
		TenantID:       lc.TenantID,
		ClientID:       lc.ClientID,
		ClientSecret:   lc.Secret,
		LoginEndpoint:  lc.LoginEndpoint,
		LoginScopes:    lc.LoginScopes,
		APIEndpoint:    lc.APIEndpoint,
		Dataset:        lc.Dataset,
		EnrichWith:     lc.EnrichWith,
		SelectUsers:    lc.Select.Users,
		SelectGroups:   lc.Select.Groups,
		SelectDevices:  lc.Select.Devices,
		SyncInterval:   lc.SyncInterval,
		UpdateInterval: lc.UpdateInterval,
	}

	if err := ec.Validate(); err != nil {
		return nil, 0, 0, err
	}
	return ecentraid.New(ec), ec.SyncInterval, ec.UpdateInterval, nil
}
