// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func init() {
	/*feature.MustRegister(feature.New(
	lbmanagement.Namespace,
	"x-pack-fleet",
	NewFleetManagerPluginV2,
	feature.MakeDetails("x-pack-fleet", "", feature.Beta)))*/
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
