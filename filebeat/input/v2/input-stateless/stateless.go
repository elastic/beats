// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package stateless

import (
	"fmt"
	"runtime/debug"

	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// InputManager provides an InputManager for transient inputs, that do not store
// state in the registry or require end-to-end event acknowledgement.
type InputManager struct {
	Configure func(*conf.C) (Input, error)
}

// Input is the interface transient inputs are required to implemented.
type Input interface {
	Name() string
	Test(v2.TestContext) error
	Run(ctx v2.Context, publish Publisher) error
}

// Publisher is used by the Input to emit events.
type Publisher interface {
	Publish(beat.Event)
}

type configuredInput struct {
	input Input
}

var _ v2.InputManager = InputManager{}

// NewInputManager wraps the given configure function to create a new stateless input manager.
func NewInputManager(configure func(*conf.C) (Input, error)) InputManager {
	return InputManager{Configure: configure}
}

// Init does nothing. Init is required to fullfil the v2.InputManager interface.
func (m InputManager) Init(_ unison.Group, _ v2.Mode) error { return nil }

// Create configures a transient input and ensures that the final input can be used with
// with the filebeat input architecture.
func (m InputManager) Create(cfg *conf.C) (v2.Input, error) {
	inp, err := m.Configure(cfg)
	if err != nil {
		return nil, err
	}
	return configuredInput{inp}, nil
}

func (si configuredInput) Name() string { return si.input.Name() }

func (si configuredInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) (err error) {
	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("input panic with: %+v\n%s", v, debug.Stack())
			}
		}
	}()

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DefaultGuarantees,
	})
	if err != nil {
		return err
	}

	defer client.Close()
	return si.input.Run(ctx, client)
}

func (si configuredInput) Test(ctx v2.TestContext) error {
	return si.input.Test(ctx)
}
