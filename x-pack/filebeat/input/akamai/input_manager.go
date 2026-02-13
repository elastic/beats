// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const inputName = "akamai"

// InputManager manages the lifecycle of akamai inputs.
type InputManager struct {
	cursor *inputcursor.InputManager
	log    *logp.Logger
}

var _ v2.InputManager = InputManager{}

// NewInputManager creates a new InputManager.
func NewInputManager(log *logp.Logger, store statestore.States) InputManager {
	return InputManager{
		log: log.Named(inputName),
		cursor: &inputcursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  cursorConfigure,
		},
	}
}

// cursorConfigure configures the cursor input from the provided configuration.
func cursorConfigure(cfg *conf.C, logger *logp.Logger) ([]inputcursor.Source, inputcursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	src := &source{cfg: config}
	return []inputcursor.Source{src}, input{}, nil
}

// source represents the Akamai data source.
type source struct {
	cfg config
}

// Name returns the name of the source (used as the cursor key).
func (s *source) Name() string {
	if s.cfg.Resource == nil || s.cfg.Resource.URL == nil {
		return s.cfg.ConfigIDs
	}
	return s.cfg.Resource.URL.String() + "/siem/v1/configs/" + s.cfg.ConfigIDs
}

// Init initializes the input manager.
func (m InputManager) Init(grp unison.Group) error {
	return m.cursor.Init(grp)
}

// Create creates a new input from the provided configuration.
func (m InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	return m.cursor.Create(cfg)
}
