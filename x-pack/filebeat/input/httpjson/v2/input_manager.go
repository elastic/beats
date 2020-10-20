// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/common"
)

// inputManager wraps one stateless input manager
// and one cursor input manager. It will create one or the other
// based on the config that is passed.
type inputManager struct {
	stateless *stateless.InputManager
}

var _ v2.InputManager = inputManager{}

// Init initializes both wrapped input managers.
func (m inputManager) Init(grp unison.Group, mode v2.Mode) error {
	return m.stateless.Init(grp, mode) // multierr.Append()
}

// Create creates a cursor input manager if the config has a date cursor set up,
// otherwise it creates a stateless input manager.
func (m inputManager) Create(cfg *common.Config) (v2.Input, error) {
	var config config
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	return m.stateless.Create(cfg)
}
