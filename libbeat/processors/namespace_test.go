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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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

		cfg, _ := config.NewConfigFrom(map[string]any{
			test.name: nil,
		})

		filter, err := ns.Plugin()(cfg, logptest.NewTestingLogger(t, ""))

		assert.NoError(t, err)
		assert.NotNil(t, filter)
	}
}

func TestNamespaceRegisterFail(t *testing.T) {
	ns := NewNamespace()
	err := ns.Register("test", newTestFilterRule)
	fatalError(t, err)

	err = ns.Register("test", newTestFilterRule)
	assert.NoError(t, err)
}

func TestNamespaceError(t *testing.T) {
	tests := []struct {
		title   string
		factory Constructor
		config  any
	}{
		{
			"no module configured",
			newTestFilterRule,
			map[string]any{},
		},
		{
			"unknown module configured",
			newTestFilterRule,
			map[string]any{
				"notTest": nil,
			},
		},
		{
			"too many modules",
			newTestFilterRule,
			map[string]any{
				"a":    nil,
				"b":    nil,
				"test": nil,
			},
		},
		{
			"filter init fail",
			func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
				return nil, errors.New("test")
			},
			map[string]any{
				"test": nil,
			},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		ns := NewNamespace()
		err := ns.Register("test", test.factory)
		fatalError(t, err)

		config, err := config.NewConfigFrom(test.config)
		fatalError(t, err)

		_, err = ns.Plugin()(config, logptest.NewTestingLogger(t, ""))
		assert.Error(t, err)
	}
}

func TestGetConstructor(t *testing.T) {
	t.Run("returns constructor for simple name", func(t *testing.T) {
		ns := NewNamespace()
		err := ns.Register("test", newTestFilterRule)
		fatalError(t, err)

		c, err := ns.GetConstructor("test")
		assert.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("returns constructor for namespaced path", func(t *testing.T) {
		ns := NewNamespace()
		err := ns.Register("a.b.c", newTestFilterRule)
		fatalError(t, err)

		c, err := ns.GetConstructor("a.b.c")
		assert.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("returns error when leaf key does not exist", func(t *testing.T) {
		ns := NewNamespace()
		err := ns.Register("test", newTestFilterRule)
		fatalError(t, err)

		_, err = ns.GetConstructor("missing")
		assert.Error(t, err)
	})

	t.Run("returns error when intermediate namespace key does not exist", func(t *testing.T) {
		ns := NewNamespace()

		_, err := ns.GetConstructor("a.b.c")
		assert.Error(t, err)
	})

	t.Run("returns error when intermediate key is a plugin not a namespace", func(t *testing.T) {
		ns := NewNamespace()
		err := ns.Register("a", newTestFilterRule)
		fatalError(t, err)

		_, err = ns.GetConstructor("a.b")
		assert.Error(t, err)
	})
}

func newTestFilterRule(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
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
