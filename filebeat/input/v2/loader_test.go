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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type loaderConfig struct {
	Plugins     []Plugin
	TypeField   string
	DefaultType string
}

type inputCheck func(t *testing.T, input Input, err error)

func TestLoader_New(t *testing.T) {
	cases := map[string]struct {
		setup loaderConfig
		check func(*testing.T, error)
	}{
		"ok": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil)},
					{Name: "b", Stability: feature.Stable, Manager: ConfigureWith(nil)},
					{Name: "c", Stability: feature.Stable, Manager: ConfigureWith(nil)},
				},
			},
			check: expectNoError,
		},
		"duplicate": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil)},
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil)},
				},
			},
			check: expectError,
		},
		"fail with invalid plugin": {
			setup: loaderConfig{
				Plugins: []Plugin{{Name: "", Manager: nil}},
			},
			check: expectError,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := test.setup.NewLoader()
			test.check(t, err)
		})
	}
}

func TestLoader_Init(t *testing.T) {
	pluginWithInit := func(name string, fn func(Mode) error) Plugin {
		return Plugin{
			Name:      name,
			Stability: feature.Stable,
			Manager:   &fakeInputManager{OnInit: fn},
		}
	}

	t.Run("calls all input managers", func(t *testing.T) {
		count := 0
		incCountOnInit := func(_ Mode) error { count++; return nil }

		setup := loaderConfig{
			Plugins: []Plugin{
				pluginWithInit("a", incCountOnInit),
				pluginWithInit("b", incCountOnInit),
			},
		}
		loader := setup.MustNewLoader()
		err := loader.Init(nil, ModeRun)
		expectNoError(t, err)
		if count != 2 {
			t.Errorf("expected init count 2, but got %v", count)
		}
	})

	t.Run("stop init on error", func(t *testing.T) {
		count := 0
		incCountOnInit := func(_ Mode) error { count++; return errors.New("oops") }
		setup := loaderConfig{
			Plugins: []Plugin{
				pluginWithInit("a", incCountOnInit),
				pluginWithInit("b", incCountOnInit),
			},
		}
		loader := setup.MustNewLoader()
		err := loader.Init(nil, ModeRun)
		expectError(t, err)
		if count != 1 {
			t.Errorf("expected init count 1, but got %v", count)
		}
	})
}

func TestLoader_Configure(t *testing.T) {
	createManager := func(name string) InputManager {
		return ConfigureWith(makeConfigFakeInput(fakeInput{Type: name}))
	}
	createPlugin := func(name string) Plugin {
		return Plugin{Name: name, Stability: feature.Stable, Manager: createManager(name)}
	}
	plugins := []Plugin{
		createPlugin("a"),
		createPlugin("b"),
		createPlugin("c"),
	}
	defaultSetup := loaderConfig{Plugins: plugins, TypeField: "type"}

	cases := map[string]struct {
		setup  loaderConfig
		config map[string]interface{}
		check  inputCheck
	}{
		"success": {
			setup:  defaultSetup,
			config: map[string]interface{}{"type": "a"},
			check:  okSetup,
		},
		"load default": {
			setup:  defaultSetup.WithDefaultType("a"),
			config: map[string]interface{}{},
			check:  okSetup,
		},
		"type is missing": {
			setup:  defaultSetup,
			config: map[string]interface{}{},
			check:  failSetup,
		},
		"unknown type": {
			setup:  defaultSetup,
			config: map[string]interface{}{"type": "unknown"},
			check:  failSetup,
		},
		"input config fails": {
			setup: defaultSetup.WithPlugins(Plugin{
				Name:      "a",
				Stability: feature.Beta,
				Manager: ConfigureWith(func(_ *common.Config) (Input, error) {
					return nil, errors.New("oops")
				}),
			}),
			config: map[string]interface{}{"type": "a"},
			check:  failSetup,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			loader := test.setup.MustNewLoader()
			input, err := loader.Configure(common.MustNewConfigFrom(test.config))
			test.check(t, input, err)
		})
	}
}

func (b loaderConfig) MustNewLoader() *Loader {
	l, err := b.NewLoader()
	if err != nil {
		panic(err)
	}
	return l
}

func (b loaderConfig) NewLoader() (*Loader, error) {
	return NewLoader(logp.NewLogger("test"), b.Plugins, b.TypeField, b.DefaultType)
}
func (b loaderConfig) WithPlugins(p ...Plugin) loaderConfig     { b.Plugins = p; return b }
func (b loaderConfig) WithTypeField(name string) loaderConfig   { b.TypeField = name; return b }
func (b loaderConfig) WithDefaultType(name string) loaderConfig { b.DefaultType = name; return b }

func failSetup(t *testing.T, _ Input, err error) {
	expectError(t, err)
}

func okSetup(t *testing.T, _ Input, err error) {
	expectNoError(t, err)
}
