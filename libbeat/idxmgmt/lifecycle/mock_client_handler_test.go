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

package lifecycle

import (
	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	mock.Mock
	cfg        LifecycleConfig
	testPolicy Policy
	mode       Mode
}

type onCall struct {
	name    string
	args    []interface{}
	returns []interface{}
}

func (c onCall) Return(values ...interface{}) onCall {
	c.returns = values
	return c
}

func newMockHandler(cfg LifecycleConfig, testPolicy Policy, calls ...onCall) *mockHandler {
	m := &mockHandler{cfg: cfg}
	for _, c := range calls {
		m.On(c.name, c.args...).Return(c.returns...)
	}
	return m
}

func onCheckEnabled() onCall { return makeOnCall("CheckEnabled") }
func (h *mockHandler) CheckEnabled() (bool, error) {
	args := h.Called()
	return args.Bool(0), args.Error(1)
}

func onHasPolicy() onCall { return makeOnCall("HasPolicy") }
func (h *mockHandler) HasPolicy() (bool, error) {
	args := h.Called()
	return args.Bool(0), args.Error(1)
}

func onCreatePolicyFromConfig() onCall { return makeOnCall("CreatePolicyFromConfig") }
func (h *mockHandler) CreatePolicyFromConfig() error {
	args := h.Called()
	return args.Error(0)

}

func (h *mockHandler) Overwrite() bool {
	return h.cfg.ILM.Overwrite || h.cfg.DSL.Overwrite
}

func (h *mockHandler) PolicyName() string {
	return h.testPolicy.Name
}

func (h *mockHandler) Policy() Policy {
	return h.testPolicy
}

func (h *mockHandler) Mode() Mode {
	return h.mode
}

func (h *mockHandler) IsElasticsearch() bool {
	return false
}

func onCheckExists() onCall { return makeOnCall("CheckExists") }
func (h *mockHandler) CheckExists() bool {
	args := h.Called()
	return args.Bool(0)
}

func makeOnCall(name string, args ...interface{}) onCall {
	return onCall{name: name, args: args}
}
