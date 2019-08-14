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

package processors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type testFilterRule struct {
	str func() string
	run func(*beat.Event) (*beat.Event, error)
}

func TestNamespace(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"test"},
		{"test.test"},
		{"abc.def.test"},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.name)

		ns := NewNamespace()
		err := ns.Register(test.name, newTestFilterRule)
		fatalError(t, err)

		cfg, _ := common.NewConfigFrom(map[string]interface{}{
			test.name: nil,
		})

		filter, err := ns.Plugin()(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, filter)
	}
}

func TestNamespaceRegisterFail(t *testing.T) {
	ns := NewNamespace()
	err := ns.Register("test", newTestFilterRule)
	fatalError(t, err)

	err = ns.Register("test", newTestFilterRule)
	assert.Error(t, err)
}

func TestNamespaceError(t *testing.T) {
	tests := []struct {
		title   string
		factory Constructor
		config  interface{}
	}{
		{
			"no module configured",
			newTestFilterRule,
			map[string]interface{}{},
		},
		{
			"unknown module configured",
			newTestFilterRule,
			map[string]interface{}{
				"notTest": nil,
			},
		},
		{
			"too many modules",
			newTestFilterRule,
			map[string]interface{}{
				"a":    nil,
				"b":    nil,
				"test": nil,
			},
		},
		{
			"filter init fail",
			func(_ *common.Config) (Processor, error) {
				return nil, errors.New("test")
			},
			map[string]interface{}{
				"test": nil,
			},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		ns := NewNamespace()
		err := ns.Register("test", test.factory)
		fatalError(t, err)

		config, err := common.NewConfigFrom(test.config)
		fatalError(t, err)

		_, err = ns.Plugin()(config)
		assert.Error(t, err)
	}
}

func newTestFilterRule(_ *common.Config) (Processor, error) {
	return &testFilterRule{}, nil
}

func (r *testFilterRule) String() string {
	if r.str == nil {
		return "test"
	}
	return r.str()
}

func (r *testFilterRule) Run(evt *beat.Event) (*beat.Event, error) {
	if r.run == nil {
		return evt, nil
	}
	return r.Run(evt)
}

func fatalError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
