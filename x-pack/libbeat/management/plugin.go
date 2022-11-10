// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/elastic/beats/v7/libbeat/feature"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func init() {
	lbmanagement.Register("x-pack-fleet", NewFleetManagerPluginV2, feature.Beta)
}

// NewFleetManagerPluginV2 registers the V2 callback
func NewFleetManagerPluginV2(config *conf.C) lbmanagement.FactoryFunc {
	c := DefaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil
		}
		return NewV2AgentManager
	}

	return nil
}
