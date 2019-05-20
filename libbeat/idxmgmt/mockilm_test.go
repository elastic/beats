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

package idxmgmt

import (
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/logp"
)

type mockILMSupport struct {
	mock.Mock
}

type onCall struct {
	name    string
	args    []interface{}
	returns []interface{}
}

func makeMockILMSupport(calls ...onCall) ilm.SupportFactory {
	return func(_ *logp.Logger, _ beat.Info, _ *common.Config) (ilm.Supporter, error) {
		m := &mockILMSupport{}
		for _, c := range calls {
			m.On(c.name, c.args...).Return(c.returns...)
		}
		return m, nil
	}
}

func (c onCall) Return(values ...interface{}) onCall {
	c.returns = values
	return c
}

func onMode() onCall { return makeOnCall("Mode") }
func (m *mockILMSupport) Mode() ilm.Mode {
	args := m.Called()
	return args.Get(0).(ilm.Mode)
}

func onAlias() onCall { return makeOnCall("Alias") }
func (m *mockILMSupport) Alias() ilm.Alias {
	args := m.Called()
	return args.Get(0).(ilm.Alias)
}

func onPolicy() onCall { return makeOnCall("Policy") }
func (m *mockILMSupport) Policy() ilm.Policy {
	args := m.Called()
	return args.Get(0).(ilm.Policy)
}

func (m *mockILMSupport) Manager(_ ilm.ClientHandler) ilm.Manager {
	return m
}

func onEnabled() onCall { return makeOnCall("Enabled") }
func (m *mockILMSupport) Enabled() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func onEnsureAlias() onCall { return makeOnCall("EnsureAlias") }
func (m *mockILMSupport) EnsureAlias() error {
	args := m.Called()
	return args.Error(0)
}

func onEnsurePolicy() onCall { return makeOnCall("EnsurePolicy") }
func (m *mockILMSupport) EnsurePolicy(overwrite bool) (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func makeOnCall(name string, args ...interface{}) onCall {
	return onCall{name: name, args: args}
}
