// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management"
	xmanagement "github.com/elastic/beats/v7/x-pack/libbeat/management"
)

func init() {
	management.Register("x-pack-fleet", NewFleetManagerPlugin, feature.Beta)
}

// NewFleetManagerPlugin creates a plugin function returning factory if configuration matches the criteria
func NewFleetManagerPlugin(config *common.Config) management.FactoryFunc {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil
		}

		if c.Mode == xmanagement.ModeFleet {
			return NewFleetManager
		}
	}

	return nil
}
