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

package v2

import (
	"errors"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/go-concert/unison"
)

type fakeInputManager struct {
	OnInit      func(Mode) error
	OnConfigure func(*common.Config) (Input, error)
}

type fakeInput struct {
	Type   string
	OnTest func(TestContext) error
	OnRun  func(Context, beat.PipelineConnector) error
}

func makeConfigFakeInput(prototype fakeInput) func(*common.Config) (Input, error) {
	return func(cfg *common.Config) (Input, error) {
		tmp := prototype
		return &tmp, nil
	}
}

func (m *fakeInputManager) Init(_ unison.Group, mode Mode) error {
	if m.OnInit != nil {
		return m.OnInit(mode)
	}
	return nil
}

func (m *fakeInputManager) Create(cfg *common.Config) (Input, error) {
	if m.OnConfigure != nil {
		return m.OnConfigure(cfg)
	}
	return nil, errors.New("oops")
}

func (f *fakeInput) Name() string { return f.Type }
func (f *fakeInput) Test(ctx TestContext) error {
	if f.OnTest != nil {
		return f.OnTest(ctx)
	}
	return nil
}

func (f *fakeInput) Run(ctx Context, pipeline beat.PipelineConnector) error {
	if f.OnRun != nil {
		return f.OnRun(ctx, pipeline)
	}
	return nil
}

func expectError(t *testing.T, err error) {
	if err == nil {
		t.Errorf("expected error")
	}
}

func expectNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
