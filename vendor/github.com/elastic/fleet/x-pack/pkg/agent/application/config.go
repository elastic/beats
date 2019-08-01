// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"errors"
	"fmt"

	"github.com/elastic/fleet/x-pack/pkg/config"
)

// Errors returned when unpacking the configuration.
var (
	ErrMissingInputType = errors.New("input type must be defined")
)

// Config define the configuration of the Agent.
type Config struct {
	Inputs []*Input `config:"inputs"`
}

// Input defines an agent input, most the configuration are delegated.
type Input struct {
	// Type is the the combination of the DataType and the inputs,
	// Example: "log/docker" generates filebeat with the docker input.
	Type string

	// RawDelegateConfig is the configuration that will be send to the transpilers.
	RawDelegateConfig *config.Config
}

// Unpack unpacks an input and keep the remaining fields in a RawDelegateConfig so they can
// be validated or transformed later.
func (i *Input) Unpack(cfg *config.RawConfig) error {
	// Keep the raw config so we can unpack it later on.
	raw, err := config.NewConfigFrom(cfg)
	if err != nil {
		return err
	}
	t := struct {
		Type string `config:"type"`
	}{}

	if err := raw.Unpack(&t); err != nil {
		return err
	}

	i.Type = t.Type
	i.RawDelegateConfig = raw

	return nil
}

// Validate validates the presence of an input type.
func (i *Input) Validate() error {
	if len(i.Type) == 0 {
		return ErrMissingInputType
	}
	return nil
}

func (i *Input) String() string {
	return fmt.Sprintf("input type is %s", i.Type)
}

// LocalDefaultConfig returns the default configuration for the agent.
func LocalDefaultConfig() *Config {
	return &Config{}
}
