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

package inputest

import (
	"errors"

	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/go-concert/unison"
)

// MockInputManager can be used as InputManager replacement in tests that require a new Input Manager.
// The OnInit and OnConfigure functions are executed if the corresponding methods get called.
type MockInputManager struct {
	OnInit      func(v2.Mode) error
	OnConfigure InputConfigurer
}

// InputConfigurer describes the interface for user supplied functions, that is
// used to create a new input from a configuration object.
type InputConfigurer func(*common.Config) (v2.Input, error)

// MockInput can be used as an Input instance in tests that require a new Input with definable behavior.
// The OnTest and OnRun functions are executed if the corresponding methods get called.
type MockInput struct {
	Type   string
	OnTest func(v2.TestContext) error
	OnRun  func(v2.Context, beat.PipelineConnector) error
}

// Init returns nil if OnInit is not set. Otherwise the return value of OnInit is returned.
func (m *MockInputManager) Init(_ unison.Group, mode v2.Mode) error {
	if m.OnInit != nil {
		return m.OnInit(mode)
	}
	return nil
}

// Create fails with an error if OnConfigure is not set. Otherwise the return
// values of OnConfigure are returned.
func (m *MockInputManager) Create(cfg *common.Config) (v2.Input, error) {
	if m.OnConfigure != nil {
		return m.OnConfigure(cfg)
	}
	return nil, errors.New("oops, OnConfigure not implemented ")
}

// Name return the `Type` field of MockInput. It is required to satisfy the v2.Input interface.
func (f *MockInput) Name() string { return f.Type }

// Test return nil if OnTest is not set. Otherwise OnTest will be called.
func (f *MockInput) Test(ctx v2.TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

// Run returns nil if OnRun is not set.
func (f *MockInput) Run(ctx v2.Context, pipeline beat.PipelineConnector) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, pipeline)
	}
	return nil
}

// ConstInputManager create a MockInputManager that always returns input when
// Configure is called. Use ConstInputManager for tests that require an
// InputManager, but create only one Input instance.
func ConstInputManager(input v2.Input) *MockInputManager {
	return &MockInputManager{OnConfigure: ConfigureConstInput(input)}
}

// ConfigureConstInput return an InputConfigurer that returns always input when called.
func ConfigureConstInput(input v2.Input) InputConfigurer {
	return func(_ *common.Config) (v2.Input, error) {
		return input, nil
	}
}

// SinglePlugin wraps an InputManager into a slice of v2.Plugin, that can be used directly with v2.NewLoader.
func SinglePlugin(name string, manager v2.InputManager) []v2.Plugin {
	return []v2.Plugin{{
		Name:      name,
		Stability: feature.Stable,
		Manager:   manager,
	}}
}
