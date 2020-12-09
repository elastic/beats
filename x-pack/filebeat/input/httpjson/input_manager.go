// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"go.uber.org/multierr"

	"github.com/elastic/go-concert/unison"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	v2 "github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/internal/v2"
)

// inputManager wraps one stateless input manager
// and one cursor input manager. It will create one or the other
// based on the config that is passed.
type inputManager struct {
	stateless *stateless.InputManager
	cursor    *cursor.InputManager

	v2inputManager v2.InputManager
}

var _ inputv2.InputManager = inputManager{}

// Init initializes both wrapped input managers.
func (m inputManager) Init(grp unison.Group, mode inputv2.Mode) error {
	return multierr.Append(
		multierr.Append(
			m.stateless.Init(grp, mode),
			m.cursor.Init(grp, mode),
		),
		m.v2inputManager.Init(grp, mode),
	)
}

// Create creates a cursor input manager if the config has a date cursor set up,
// otherwise it creates a stateless input manager.
func (m inputManager) Create(cfg *common.Config) (inputv2.Input, error) {
	if v, _ := cfg.String("config_version", -1); v == "2" {
		return m.v2inputManager.Create(cfg)
	}
	cfgwarn.Deprecate("7.12", "you are using a deprecated version of httpjson config")
	config := newDefaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if config.DateCursor != nil {
		return m.cursor.Create(cfg)
	}

	return m.stateless.Create(cfg)
}
