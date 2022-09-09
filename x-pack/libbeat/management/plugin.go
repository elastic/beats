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
	lbmanagement.Register("x-pack-fleet", NewFleetManagerPlugin, feature.Beta)
}

// NewFleetManagerPlugin creates a plugin function returning factory if configuration matches the criteria
func NewFleetManagerPlugin(config *conf.C) lbmanagement.FactoryFunc {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil
		}
		return NewFleetManager
	}

	return nil
}
