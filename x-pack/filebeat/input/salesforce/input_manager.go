// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"fmt"
	"time"

	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

// compile-time check if querier implements InputManager
var _ v2.InputManager = InputManager{}

// InputManager wraps one stateless input manager
// and one cursor input manager. It will create one or the other
// based on the config that is passed.
type InputManager struct {
	cursor *inputcursor.InputManager
}

// NewInputManager creates a new input manager.
func NewInputManager(log *logp.Logger, store inputcursor.StateStore) InputManager {
	return InputManager{
		cursor: &inputcursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  cursorConfigure,
		},
	}
}

func defaultConfig() config {
	apiVersion := 58
	maxAttempts := 5
	waitMin := time.Second
	waitMax := time.Minute
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 30 * time.Second

	return config{
		Version: apiVersion,
		Resource: &resourceConfig{
			Transport: transport,
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
		},
	}
}

// cursorConfigure configures the cursor input manager.
func cursorConfigure(cfg *conf.C) ([]inputcursor.Source, inputcursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, fmt.Errorf("reading config: %w", err)
	}
	sources := []inputcursor.Source{&source{cfg: config}}
	return sources, &salesforceInput{config: config}, nil
}

type source struct{ cfg config }

func (s *source) Name() string { return s.cfg.URL }

// Init initializes both wrapped input managers.
func (m InputManager) Init(grp unison.Group, mode v2.Mode) error {
	return m.cursor.Init(grp, mode)
}

// Create creates a cursor input manager.
func (m InputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	return m.cursor.Create(cfg)
}
