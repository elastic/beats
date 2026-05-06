// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const inputName = "akamai"

// InputManager manages the lifecycle of akamai inputs.
type InputManager struct {
	store statestore.States
	log   *logp.Logger
}

var _ v2.InputManager = InputManager{}

// NewInputManager creates a new InputManager.
func NewInputManager(log *logp.Logger, store statestore.States) InputManager {
	return InputManager{
		log:   log.Named(inputName),
		store: store,
	}
}

// Init initializes the input manager.
func (m InputManager) Init(_ unison.Group) error {
	return nil
}

// Create creates a new input from the provided configuration.
func (m InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	return &akamaiInput{cfg: config, store: m.store}, nil
}
