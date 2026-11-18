// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	ecentraid "github.com/elastic/entcollect/provider/entraid"
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
	if mc.expand {
		log.Warn("expand options are not supported with use_minimal_state and are ignored")
	}
	client, err := mc.transport.Client(httpcommon.WithLogger(log), httpcommon.WithAPMHTTPInstrumentation())
	if err != nil {
		return provider.MinimalState{}, fmt.Errorf("creating HTTP client: %w", err)
	}
	return provider.MinimalState{
		Provider:        ecentraid.NewWithClient(mc.provider, client),
		FullSync:        mc.provider.SyncInterval,
		IncrementalSync: mc.provider.UpdateInterval,
		Client:          client,
		Tracer:          mc.tracer,
		// Token requests carry the client secret and return the access
		// token, so keep them out of the trace, as the legacy provider
		// does by using a separate client for them.
		Untraced: []string{strings.TrimRight(mc.provider.LoginEndpoint, "/") + "/"},
	}, nil
}

// minimalConf is the minimal-state configuration of the Entra ID provider.
type minimalConf struct {
	provider  ecentraid.Config
	transport httpcommon.HTTPTransportSettings
	tracer    *provider.TracerConfig

	// expand is whether any expand options were set. They cannot be
	// applied: the user and device delta queries do not support
	// $expand, and the provider does not expand groups.
	expand bool
}

// minimalConfig unpacks cfg into a validated minimalConf.
func minimalConfig(cfg *config.C) (minimalConf, error) {
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
		Expand *config.C `config:"expand"`

		SyncInterval   time.Duration `config:"sync_interval"`
		UpdateInterval time.Duration `config:"update_interval"`

		ScratchDir string `config:"scratch_dir"`

		// The HTTP settings are at the top level, as in the legacy
		// provider.
		Transport httpcommon.HTTPTransportSettings `config:",inline"`
		Tracer    *provider.TracerConfig           `config:"tracer"`
	}

	// The select lists are not seeded with their defaults: ucfg merges
	// a list into an existing one by index, so a shorter configured
	// list would keep the tail of the default. Empty lists get their
	// defaults after Unpack.
	d := ecentraid.DefaultConfig()
	lc := localConf{
		LoginEndpoint:  d.LoginEndpoint,
		LoginScopes:    d.LoginScopes,
		APIEndpoint:    d.APIEndpoint,
		Dataset:        d.Dataset,
		EnrichWith:     d.EnrichWith,
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
		Transport:      httpcommon.DefaultHTTPTransportSettings(),
	}
	if err := cfg.Unpack(&lc); err != nil {
		return minimalConf{}, err
	}
	if len(lc.Select.Users) == 0 {
		lc.Select.Users = d.SelectUsers
	}
	if len(lc.Select.Groups) == 0 {
		lc.Select.Groups = d.SelectGroups
	}
	if len(lc.Select.Devices) == 0 {
		lc.Select.Devices = d.SelectDevices
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
		ScratchDir:     lc.ScratchDir,
	}
	if err := ec.Validate(); err != nil {
		return minimalConf{}, err
	}
	return minimalConf{
		provider:  ec,
		transport: lc.Transport,
		tracer:    lc.Tracer,
		expand:    lc.Expand != nil,
	}, nil
}
