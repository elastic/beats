// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management"
)

func init() {
	management.Register("x-pack", NewManagerPlugin, feature.Beta)
}

// NewManagerPlugin creates a plugin function returning factory if configuration matches the criteria
func NewManagerPlugin(config *common.Config) management.FactoryFunc {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil
		}

		if c.Mode == ModeCentralManagement {
			return NewConfigManager
		}
	}

	return nil
}
