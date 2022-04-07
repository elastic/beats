// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
)

const (
	ScopeNode    = "node"
	ScopeCluster = "cluster"
)

type nomadAnnotatorConfig struct {
	Address         string        `config:"address"`
	Region          string        `config:"region"`
	Namespace       string        `config:"namespace"`
	SecretID        string        `config:"secret_id"`
	Node            string        `config:"node"`
	Scope           string        `config:"scope"`
	RefreshInterval time.Duration `config:"refresh_interval"`
	// Annotations are kept after the allocations is removed, until they haven't been accessed for a
	// full `cleanup_timeout`:
	CleanupTimeout  time.Duration `config:"cleanup_timeout"`
	Indexers        PluginConfig  `config:"indexers"`
	Matchers        PluginConfig  `config:"matchers"`
	DefaultMatchers Enabled       `config:"default_matchers"`
	DefaultIndexers Enabled       `config:"default_indexers"`

	syncPeriod time.Duration
}

func (c *nomadAnnotatorConfig) Validate() error {
	switch c.Scope {
	case ScopeNode:
	case ScopeCluster:
	default:
		return fmt.Errorf("invalid value for `scope`: %s, select `%s` or `%s`", c.Scope, ScopeNode, ScopeCluster)
	}
	return nil
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func defaultNomadAnnotatorConfig() nomadAnnotatorConfig {
	return nomadAnnotatorConfig{
		Address:         "http://127.0.0.1:4646",
		Region:          "",
		Namespace:       "",
		SecretID:        "",
		Scope:           ScopeNode,
		syncPeriod:      5 * time.Second,
		CleanupTimeout:  60 * time.Second,
		DefaultMatchers: Enabled{true},
		DefaultIndexers: Enabled{true},
		RefreshInterval: 30 * time.Second,
	}
}
