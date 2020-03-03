// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

type nomadAnnotatorConfig struct {
	Host            string        `config:"host"`
	Namespace       string        `config:"namespace"`
	SyncPeriod      time.Duration `config:"sync_period"`
	RefreshInterval time.Duration `config:"refresh_interval"`
	// Annotations are kept after pod is removed, until they haven't been accessed
	// for a full `cleanup_timeout`:
	CleanupTimeout  time.Duration `config:"cleanup_timeout"`
	Indexers        PluginConfig  `config:"indexers"`
	Matchers        PluginConfig  `config:"matchers"`
	DefaultMatchers Enabled       `config:"default_matchers"`
	DefaultIndexers Enabled       `config:"default_indexers"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func defaultNomadAnnotatorConfig() nomadAnnotatorConfig {
	return nomadAnnotatorConfig{
		SyncPeriod:      5 * time.Second,
		CleanupTimeout:  60 * time.Second,
		DefaultMatchers: Enabled{true},
		DefaultIndexers: Enabled{true},
		RefreshInterval: 30 * time.Second,
	}
}
