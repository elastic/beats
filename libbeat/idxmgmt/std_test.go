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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/beats/v7/libbeat/template"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type mockCreateOp uint8

const (
	mockCreatePolicy mockCreateOp = iota
	mockCreateTemplate
)

func TestDefaultSupport_Enabled(t *testing.T) {
	cases := map[string]struct {
		ilmCalls []onCall
		cfg      map[string]interface{}
		enabled  bool
	}{
		"templates and ilm disabled": {
			enabled: false,
			ilmCalls: []onCall{
				onEnabled().Return(false),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
		"templates only": {
			enabled: true,
			ilmCalls: []onCall{
				onEnabled().Return(false),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": true,
			},
		},
		"ilm only": {
			enabled: true,
			ilmCalls: []onCall{
				onEnabled().Return(true),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			info := beat.Info{Beat: "test", Version: "9.9.9"}
			factory := MakeDefaultSupport(makeMockILMSupport(test.ilmCalls...))
			im, err := factory(nil, info, config.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)
			assert.Equal(t, test.enabled, im.Enabled())
		})
	}
}

func TestDefaultSupport_BuildSelector(t *testing.T) {
	type nameFunc func(time.Time) string

	noILM := []onCall{onEnabled().Return(false)}
	ilmTemplateSettings := func(policy string) []onCall {
		return []onCall{
			onEnabled().Return(true),
			onPolicy().Return(lifecycle.Policy{Name: policy}),
		}
	}

	stable := func(s string) nameFunc {
		return func(_ time.Time) string { return s }
	}

	cases := map[string]struct {
		ilmCalls []onCall
		imCfg    map[string]interface{}
		cfg      map[string]interface{}
		want     nameFunc
		meta     mapstr.M
	}{
		"without ilm": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"without ilm must be lowercase": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "TeSt-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"event index without ilm": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test"),
			meta: mapstr.M{
				"index": "test",
			},
		},
		"event index without ilm must be lowercase": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test"),
			meta: mapstr.M{
				"index": "Test",
			},
		},
		"with ilm": {
			ilmCalls: ilmTemplateSettings("test-9.9.9"),
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"with ilm must be lowercase": {
			ilmCalls: ilmTemplateSettings("Test-9.9.9"),
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"event index with ilm": {
			ilmCalls: ilmTemplateSettings("test-9.9.9"),
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("event-index"),
			meta: mapstr.M{
				"index": "event-index",
			},
		},
		"use indices": {
			ilmCalls: ilmTemplateSettings("test-9.9.9"),
			cfg: map[string]interface{}{
				"index": "test-%{[agent.version]}",
				"indices": []map[string]interface{}{
					{"index": "myindex"},
				},
			},
			want: stable("myindex"),
		},
		"use indices settings must be lowercase": {
			ilmCalls: ilmTemplateSettings("test-9.9.9"),
			cfg: map[string]interface{}{
				"index": "test-%{[agent.version]}",
				"indices": []map[string]interface{}{
					{"index": "MyIndex"},
				},
			},
			want: stable("myindex"),
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			ts := time.Now()
			info := beat.Info{Beat: "test", Version: "9.9.9"}

			factory := MakeDefaultSupport(makeMockILMSupport(test.ilmCalls...))
			im, err := factory(nil, info, config.MustNewConfigFrom(test.imCfg))
			require.NoError(t, err)

			sel, err := im.BuildSelector(config.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)

			meta := test.meta
			idx, err := sel.Select(&beat.Event{
				Timestamp: ts,
				Fields: mapstr.M{
					"test": "value",
					"agent": mapstr.M{
						"version": "9.9.9",
					},
				},
				Meta: meta,
			})
			require.NoError(t, err)
			assert.Equal(t, test.want(ts), idx)
		})
	}
}

func TestIndexManager_VerifySetup(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	for name, setup := range map[string]struct {
		tmplEnabled, ilmEnabled, ilmOverwrite bool
		loadTmpl, loadILM                     LoadMode
		lifecycle                             lifecycle.LifecycleConfig
		ok                                    bool
		warn                                  string
	}{
		"load template with ilm without loading ilm": {
			ilmEnabled: true, tmplEnabled: true, loadILM: LoadModeDisabled,
			warn:      "whithout loading ILM policy",
			lifecycle: lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: true, PolicyName: *fmtstr.MustCompileEvent("test")}},
		},
		"load ilm without template": {
			ilmEnabled: true, loadILM: LoadModeUnset,
			warn:      "without loading template is not recommended",
			lifecycle: lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: true, PolicyName: *fmtstr.MustCompileEvent("test")}},
		},
		"template disabled but loading enabled": {
			loadTmpl:  LoadModeEnabled,
			warn:      "loading not enabled",
			lifecycle: lifecycle.DefaultILMConfig(info),
		},
		"ilm disabled but loading enabled": {
			loadILM: LoadModeEnabled, tmplEnabled: true,
			warn:      "loading not enabled",
			lifecycle: lifecycle.DefaultILMConfig(info),
		},
		"ilm enabled but loading disabled": {
			ilmEnabled: true, loadILM: LoadModeDisabled,
			warn:      "loading not enabled",
			lifecycle: lifecycle.DefaultILMConfig(info),
		},
		"template enabled but loading disabled": {
			tmplEnabled: true, loadTmpl: LoadModeDisabled,
			warn:      "loading not enabled",
			lifecycle: lifecycle.DefaultILMConfig(info),
		},
		"ilm enabled but overwrite disabled": {
			tmplEnabled: true,
			ilmEnabled:  true, ilmOverwrite: false, loadILM: LoadModeEnabled,
			warn:      "Overwriting lifecycle policy is disabled",
			lifecycle: lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: true, Overwrite: false, PolicyName: *fmtstr.MustCompileEvent("test")}},
		},
		"everything enabled": {
			tmplEnabled: true,
			ilmEnabled:  true, ilmOverwrite: true,
			lifecycle: lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: true, Overwrite: true, PolicyName: *fmtstr.MustCompileEvent("test")}},
			ok:        true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(mapstr.M{
				"setup.ilm.enabled":      setup.ilmEnabled,
				"setup.ilm.overwrite":    setup.ilmOverwrite,
				"setup.template.enabled": setup.tmplEnabled,
			})
			require.NoError(t, err)
			support, err := MakeDefaultSupport(lifecycle.StdSupport)(nil, beat.Info{}, cfg)
			require.NoError(t, err)
			clientHandler, err := newMockClientHandler(setup.lifecycle, info)
			require.NoError(t, err)
			manager := support.Manager(clientHandler, nil)
			ok, warn := manager.VerifySetup(setup.loadTmpl, setup.loadILM)
			assert.Equal(t, setup.ok, ok)
			assert.Contains(t, warn, setup.warn)
			clientHandler.assertInvariants(t)
		})
	}
}

func TestIndexManager_Setup(t *testing.T) {
	cloneCfg := func(c template.TemplateConfig) template.TemplateConfig {
		if c.AppendFields != nil {
			tmp := make(mapping.Fields, len(c.AppendFields))
			copy(tmp, c.AppendFields)
			c.AppendFields = tmp
		}

		if c.Settings.Index != nil {
			c.Settings.Index = (map[string]interface{})(mapstr.M(c.Settings.Index).Clone())
		}
		if c.Settings.Source != nil {
			c.Settings.Source = (map[string]interface{})(mapstr.M(c.Settings.Source).Clone())
		}
		return c
	}

	cfgWith := func(s template.TemplateConfig, mods ...map[string]interface{}) *template.TemplateConfig {
		for _, mod := range mods {
			cfg := config.MustNewConfigFrom(mod)
			s = cloneCfg(s)
			err := cfg.Unpack(&s)
			if err != nil {
				panic(err)
			}
			if s.Settings.Index != nil && len(s.Settings.Index) == 0 {
				s.Settings.Index = nil
			}
			if s.Settings.Source != nil && len(s.Settings.Source) == 0 {
				s.Settings.Source = nil
			}
		}
		return &s
	}
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	defaultCfg := template.DefaultConfig(info)
	defaultLifecycleConfig := lifecycle.DefaultILMConfig(info)
	dslLifecycleConfig := lifecycle.DefaultDSLConfig(info)
	cases := map[string]struct {
		cfg                   mapstr.M
		loadTemplate, loadILM LoadMode
		ilmCfg                lifecycle.LifecycleConfig
		err                   bool
		tmplCfg               *template.TemplateConfig
		policy                string
	}{
		"template default ilm default": {
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9",
				"settings.index.lifecycle.name": "test",
			}),
			policy: "test",
			ilmCfg: defaultLifecycleConfig,
		},
		"template-default-dsl-config": {
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9",
				"settings.index.lifecycle.name": "test-9.9.9",
			}),
			policy: "test-9.9.9",
			ilmCfg: dslLifecycleConfig,
		},
		"template default ilm default with policy changed": {
			cfg: mapstr.M{
				"setup.ilm.policy_name": "policy-keep",
			},
			ilmCfg: lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: true, CheckExists: true, PolicyName: *fmtstr.MustCompileEvent("policy-keep")}},
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9",
				"settings.index.lifecycle.name": "policy-keep",
			}),
			policy: "policy-keep",
		},
		"template default ilm disabled": {
			cfg: mapstr.M{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeEnabled,
			ilmCfg:       lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			tmplCfg:      &defaultCfg,
		},
		"template default loadMode Overwrite ilm disabled": {
			cfg: mapstr.M{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeOverwrite,
			ilmCfg:       lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"overwrite": "true",
				"name":      "test-9.9.9",
				"pattern":   "test-9.9.9",
			}),
		},
		"template default loadMode Force ilm disabled": {
			cfg: mapstr.M{
				"setup.ilm.enabled": false,
				"name":              "test-9.9.9",
				"pattern":           "test-9.9.9",
			},
			loadTemplate: LoadModeForce,
			ilmCfg:       lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"overwrite": "true",
			}),
		},
		"template loadMode disabled ilm disabled": {
			cfg: mapstr.M{
				"setup.ilm.enabled": false,
			},
			ilmCfg:       lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			loadTemplate: LoadModeDisabled,
		},
		"template disabled ilm default": {
			cfg: mapstr.M{
				"setup.template.enabled": false,
			},
			ilmCfg: defaultLifecycleConfig,
			policy: "test",
		},
		"template disabled ilm disabled, loadMode Overwrite": {
			cfg: mapstr.M{
				"setup.template.enabled": false,
				"setup.ilm.enabled":      false,
			},
			ilmCfg:  lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			loadILM: LoadModeOverwrite,
		},
		"template disabled ilm disabled loadMode Force": {
			cfg: mapstr.M{
				"setup.template.enabled": false,
				"setup.ilm.enabled":      false,
			},
			ilmCfg:  lifecycle.LifecycleConfig{ILM: lifecycle.Config{Enabled: false}},
			loadILM: LoadModeForce,
		},
		"template loadmode disabled ilm loadMode enabled": {
			loadTemplate: LoadModeDisabled,
			loadILM:      LoadModeEnabled,
			ilmCfg:       defaultLifecycleConfig,
			policy:       "test",
		},
		"template default ilm loadMode disabled": {
			loadILM: LoadModeDisabled,
			ilmCfg:  defaultLifecycleConfig,
			policy:  "test",
			tmplCfg: cfgWith(template.DefaultConfig(info), map[string]interface{}{
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9",
				"settings.index.lifecycle.name": "test",
			}),
		},
		"template loadmode disabled ilm loadmode disabled": {
			loadTemplate: LoadModeDisabled,
			ilmCfg:       defaultLifecycleConfig,
			loadILM:      LoadModeDisabled,
			policy:       "test",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			factory := MakeDefaultSupport(lifecycle.StdSupport)
			im, err := factory(nil, info, config.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)

			clientHandler, err := newMockClientHandler(test.ilmCfg, info)
			require.NoError(t, err)
			manager := im.Manager(clientHandler, BeatsAssets([]byte("testbeat fields")))
			err = manager.Setup(test.loadTemplate, test.loadILM)
			clientHandler.assertInvariants(t)
			if test.err {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if test.tmplCfg == nil {
					assert.Nil(t, clientHandler.tmplCfg)

				} else {
					assert.Equal(t, test.tmplCfg, clientHandler.tmplCfg)
				}
				assert.Equal(t, test.policy, clientHandler.policyName)
			}
		})
	}
}

func (op mockCreateOp) String() string {
	names := []string{"create-policy", "create-template"}
	if int(op) > len(names) {
		return "unknown"
	}
	return names[op]
}

type mockClientHandler struct {
	policyName      string
	installedPolicy bool

	tmplCfg     *template.TemplateConfig
	tmplForce   bool
	lifecycle   lifecycle.LifecycleConfig
	selectedCfg lifecycle.Config
	operations  []mockCreateOp
	mode        lifecycle.Mode
}

func newMockClientHandler(cfg lifecycle.LifecycleConfig, info beat.Info) (*mockClientHandler, error) {
	if cfg.ILM.Enabled && cfg.DSL.Enabled {
		return nil, errors.New("both ILM and DSL enabled")
	}

	selectedCfg := cfg.ILM
	if cfg.DSL.Enabled {
		selectedCfg = cfg.DSL
	}

	var name string
	var err error
	if selectedCfg.Enabled {
		name, err = lifecycle.ApplyStaticFmtstr(info, selectedCfg.PolicyName)
		if err != nil {
			return nil, fmt.Errorf("error applying formatting string for template name: %w", err)
		}
	}

	return &mockClientHandler{selectedCfg: selectedCfg, policyName: name}, nil
}

func (h *mockClientHandler) Load(config template.TemplateConfig, _ beat.Info, fields []byte, migration bool) error {
	h.recordOp(mockCreateTemplate)
	h.tmplForce = config.Overwrite
	h.tmplCfg = &config
	return nil
}

func (h *mockClientHandler) CheckEnabled() (bool, error) {
	return h.selectedCfg.Enabled, nil
}

func (h *mockClientHandler) CheckExists() bool {
	return h.selectedCfg.CheckExists
}

func (h *mockClientHandler) Overwrite() bool {
	return h.selectedCfg.Overwrite
}

func (h *mockClientHandler) HasPolicy() (bool, error) {
	return h.installedPolicy, nil
}

func (h *mockClientHandler) PolicyName() string {
	return h.policyName
}

func (h *mockClientHandler) Policy() lifecycle.Policy {
	return lifecycle.Policy{}
}

func (h *mockClientHandler) Mode() lifecycle.Mode {
	return h.mode
}

func (h *mockClientHandler) IsElasticsearch() bool {
	return true
}

func (h *mockClientHandler) createILMPolicy(policy lifecycle.Policy) error {
	h.recordOp(mockCreatePolicy)
	h.policyName = policy.Name
	return nil
}

func (h *mockClientHandler) CreatePolicyFromConfig() error {
	h.installedPolicy = true
	h.recordOp(mockCreatePolicy)
	if h.lifecycle.DSL.Enabled {
		return h.createILMPolicy(lifecycle.Policy{Name: h.policyName, Body: lifecycle.DefaultILMPolicy})
	}
	return h.createILMPolicy(lifecycle.Policy{Name: h.policyName, Body: lifecycle.DefaultDSLPolicy})
}

func (h *mockClientHandler) recordOp(op mockCreateOp) {
	h.operations = append(h.operations, op)
}

func (h *mockClientHandler) assertInvariants(t *testing.T) {
	for i, op := range h.operations {
		for _, older := range h.operations[:i] {
			if older > op {
				t.Errorf("Operation: '%v' has been executed before '%v'", older, op)
			}
		}
	}
}
