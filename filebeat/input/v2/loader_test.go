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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/version"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil, logp.NewNopLogger())},
					{Name: "b", Stability: feature.Stable, Manager: ConfigureWith(nil, logp.NewNopLogger())},
					{Name: "c", Stability: feature.Stable, Manager: ConfigureWith(nil, logp.NewNopLogger())},
				},
			},
			check: expectNoError,
		},
		"duplicate": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil, logp.NewNopLogger())},
					{Name: "a", Stability: feature.Stable, Manager: ConfigureWith(nil, logp.NewNopLogger())},
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
	pluginWithInit := func(name string, fn func() error) Plugin {
		return Plugin{
			Name:      name,
			Stability: feature.Stable,
			Manager:   &fakeInputManager{OnInit: fn},
		}
	}

	t.Run("calls all input managers", func(t *testing.T) {
		count := 0
		incCountOnInit := func() error { count++; return nil }

		setup := loaderConfig{
			Plugins: []Plugin{
				pluginWithInit("a", incCountOnInit),
				pluginWithInit("b", incCountOnInit),
			},
		}
		loader := setup.MustNewLoader()
		err := loader.Init(nil)
		expectNoError(t, err)
		if count != 2 {
			t.Errorf("expected init count 2, but got %v", count)
		}
	})

	t.Run("stop init on error", func(t *testing.T) {
		count := 0
		incCountOnInit := func() error { count++; return errors.New("oops") }
		setup := loaderConfig{
			Plugins: []Plugin{
				pluginWithInit("a", incCountOnInit),
				pluginWithInit("b", incCountOnInit),
			},
		}
		loader := setup.MustNewLoader()
		err := loader.Init(nil)
		expectError(t, err)
		if count != 1 {
			t.Errorf("expected init count 1, but got %v", count)
		}
	})
}

func TestLoader_Configure(t *testing.T) {
	createManager := func(name string) InputManager {
		return ConfigureWith(makeConfigFakeInput(fakeInput{Type: name}), logp.NewNopLogger())
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
				Manager: ConfigureWith(func(_ *conf.C, _ *logp.Logger) (Input, error) {
					return nil, errors.New("oops")
				}, logp.NewNopLogger()),
			}),
			config: map[string]interface{}{"type": "a"},
			check:  failSetup,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			loader := test.setup.MustNewLoader()
			input, err := loader.Configure(conf.MustNewConfigFrom(test.config))
			test.check(t, input, err)
		})
	}
}

func TestLoader_ConfigureRedirect(t *testing.T) {
	targetManager := ConfigureWith(
		makeConfigFakeInput(fakeInput{Type: "target"}),
		logp.NewNopLogger(),
	)

	cases := map[string]struct {
		setup  loaderConfig
		config map[string]interface{}
		check  inputCheck
	}{
		"redirect_success": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{
						Name:      "source",
						Stability: feature.Stable,
						Manager: &fakeRedirectManager{
							fakeInputManager: fakeInputManager{
								OnConfigure: func(_ *conf.C) (Input, error) {
									return nil, errors.New("should not be called")
								},
							},
							OnRedirect: func(cfg *conf.C) (string, *conf.C, error) {
								return "target", cfg, nil
							},
						},
					},
					{Name: "target", Stability: feature.Stable, Manager: targetManager},
				},
				TypeField: "type",
			},
			config: map[string]interface{}{"type": "source"},
			check:  okSetup,
		},
		"no_redirect_passthrough": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{
						Name:      "source",
						Stability: feature.Stable,
						Manager: &fakeRedirectManager{
							fakeInputManager: fakeInputManager{
								OnConfigure: func(_ *conf.C) (Input, error) {
									return &fakeInput{Type: "source"}, nil
								},
							},
							OnRedirect: func(_ *conf.C) (string, *conf.C, error) {
								return "", nil, nil
							},
						},
					},
				},
				TypeField: "type",
			},
			config: map[string]interface{}{"type": "source"},
			check:  okSetup,
		},
		"redirect_to_unknown_target": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{
						Name:      "source",
						Stability: feature.Stable,
						Manager: &fakeRedirectManager{
							OnRedirect: func(_ *conf.C) (string, *conf.C, error) {
								return "nonexistent", conf.NewConfig(), nil
							},
						},
					},
				},
				TypeField: "type",
			},
			config: map[string]interface{}{"type": "source"},
			check:  failSetup,
		},
		"redirect_error": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{
						Name:      "source",
						Stability: feature.Stable,
						Manager: &fakeRedirectManager{
							OnRedirect: func(_ *conf.C) (string, *conf.C, error) {
								return "", nil, errors.New("translation failed")
							},
						},
					},
				},
				TypeField: "type",
			},
			config: map[string]interface{}{"type": "source"},
			check:  failSetup,
		},
		"non-redirector_manager_unchanged": {
			setup: loaderConfig{
				Plugins: []Plugin{
					{
						Name:      "plain",
						Stability: feature.Stable,
						Manager:   ConfigureWith(makeConfigFakeInput(fakeInput{Type: "plain"}), logp.NewNopLogger()),
					},
				},
				TypeField: "type",
			},
			config: map[string]interface{}{"type": "plain"},
			check:  okSetup,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			loader := test.setup.MustNewLoader()
			input, err := loader.Configure(conf.MustNewConfigFrom(test.config))
			test.check(t, input, err)
		})
	}
}

func TestLoader_DeleteRedirect(t *testing.T) {
	t.Run("delete_follows_redirect", func(t *testing.T) {
		var deletedWith *conf.C
		translatedCfg := conf.MustNewConfigFrom(map[string]interface{}{
			"type": "target",
			"key":  "translated",
		})

		setup := loaderConfig{
			Plugins: []Plugin{
				{
					Name:      "source",
					Stability: feature.Stable,
					Manager: &fakeRedirectManager{
						OnRedirect: func(_ *conf.C) (string, *conf.C, error) {
							return "target", translatedCfg, nil
						},
					},
				},
				{
					Name:      "target",
					Stability: feature.Stable,
					Manager: &fakeDeleteManager{
						fakeInputManager: fakeInputManager{
							OnConfigure: func(_ *conf.C) (Input, error) {
								return &fakeInput{Type: "target"}, nil
							},
						},
						OnDelete: func(cfg *conf.C) error {
							deletedWith = cfg
							return nil
						},
					},
				},
			},
			TypeField: "type",
		}

		loader := setup.MustNewLoader()
		err := loader.Delete(conf.MustNewConfigFrom(map[string]interface{}{"type": "source"}))
		require.NoError(t, err)
		require.NotNil(t, deletedWith, "Delete should have been called on target")

		key, err := deletedWith.String("key", -1)
		require.NoError(t, err)
		require.Equal(t, "translated", key)
	})

	t.Run("delete_no_redirect", func(t *testing.T) {
		var deleted bool
		setup := loaderConfig{
			Plugins: []Plugin{
				{
					Name:      "source",
					Stability: feature.Stable,
					Manager: &fakeDeleteManager{
						OnDelete: func(_ *conf.C) error {
							deleted = true
							return nil
						},
					},
				},
			},
			TypeField: "type",
		}

		loader := setup.MustNewLoader()
		err := loader.Delete(conf.MustNewConfigFrom(map[string]interface{}{"type": "source"}))
		require.NoError(t, err)
		require.True(t, deleted, "Delete should have been called on source")
	})

	t.Run("delete_on_non-deletable_manager_is_noop", func(t *testing.T) {
		setup := loaderConfig{
			Plugins: []Plugin{
				{
					Name:      "source",
					Stability: feature.Stable,
					Manager: &fakeRedirectManager{
						OnRedirect: func(_ *conf.C) (string, *conf.C, error) {
							return "target", conf.NewConfig(), nil
						},
					},
				},
				{
					Name:      "target",
					Stability: feature.Stable,
					Manager:   ConfigureWith(nil, logp.NewNopLogger()),
				},
			},
			TypeField: "type",
		}

		loader := setup.MustNewLoader()
		err := loader.Delete(conf.MustNewConfigFrom(map[string]interface{}{"type": "source"}))
		require.NoError(t, err)
	})
}

func TestLoader_ConfigureFIPS(t *testing.T) {
	loaderCfg := loaderConfig{
		Plugins: []Plugin{
			{
				Name:      "a",
				Stability: feature.Stable,
				Manager: ConfigureWith(func(_ *conf.C, _ *logp.Logger) (Input, error) {
					return nil, nil
				}, logp.NewNopLogger()),
				ExcludeFromFIPS: true,
			},
		},
		TypeField: "type",
	}

	loader := loaderCfg.MustNewLoader()
	input, err := loader.Configure(conf.MustNewConfigFrom(map[string]any{"type": "a"}))
	require.Nil(t, input)

	if version.FIPSDistribution {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	t.Logf("FIPS distribution = %v; err = %v", version.FIPSDistribution, err)
}

func (b loaderConfig) MustNewLoader() *Loader {
	l, err := b.NewLoader()
	if err != nil {
		panic(err)
	}
	return l
}

func (b loaderConfig) NewLoader() (*Loader, error) {
	logger, _ := logp.NewDevelopmentLogger("")
	return NewLoader(logger, b.Plugins, b.TypeField, b.DefaultType)
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
