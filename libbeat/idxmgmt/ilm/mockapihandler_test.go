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

package ilm

import (
	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	mock.Mock
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

func newMockHandler(calls ...onCall) *mockHandler {
	m := &mockHandler{}
	for _, c := range calls {
		m.On(c.name, c.args...).Return(c.returns...)
	}
	return m
}

func onCheckILMEnabled(enabled bool) onCall { return makeOnCall("CheckILMEnabled", enabled) }
func (h *mockHandler) CheckILMEnabled(enabled bool) (bool, error) {
	args := h.Called(enabled)
	return args.Bool(0), args.Error(1)
}

func onHasILMPolicy(name string) onCall { return makeOnCall("HasILMPolicy", name) }
func (h *mockHandler) HasILMPolicy(name string) (bool, error) {
	args := h.Called(name)
	return args.Bool(0), args.Error(1)
}

func onCreateILMPolicy(policy Policy) onCall { return makeOnCall("CreateILMPolicy", policy) }
func (h *mockHandler) CreateILMPolicy(policy Policy) error {
	args := h.Called(policy)
	return args.Error(0)
}

func makeOnCall(name string, args ...interface{}) onCall {
	return onCall{name: name, args: args}
}
