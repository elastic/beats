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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/mapping"
	"github.com/elastic/beats/libbeat/template"
)

type mockClientHandler struct {
	alias, policy string
	expectsPolicy bool

	tmplCfg   *template.TemplateConfig
	tmplForce bool

	operations []mockCreateOp
}

type mockCreateOp uint8

const (
	mockCreatePolicy mockCreateOp = iota
	mockCreateTemplate
	mockCreateAlias
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
				onMode().Return(ilm.ModeDisabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
		"templates only": {
			enabled: true,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeDisabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": true,
			},
		},
		"ilm only": {
			enabled: true,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeEnabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
		"ilm tentatively": {
			enabled: true,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeAuto),
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
			im, err := factory(nil, info, common.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)
			assert.Equal(t, test.enabled, im.Enabled())
		})
	}
}

func TestDefaultSupport_BuildSelector(t *testing.T) {
	type nameFunc func(time.Time) string

	noILM := []onCall{onMode().Return(ilm.ModeDisabled)}
	ilmTemplateSettings := func(alias, policy string) []onCall {
		return []onCall{
			onMode().Return(ilm.ModeEnabled),
			onAlias().Return(ilm.Alias{Name: alias}),
			onPolicy().Return(ilm.Policy{Name: policy}),
		}
	}

	stable := func(s string) nameFunc {
		return func(_ time.Time) string { return s }
	}
	dateIdx := func(base string) nameFunc {
		return func(ts time.Time) string {
			ts = ts.UTC()
			ext := fmt.Sprintf("%d.%02d.%02d", ts.Year(), ts.Month(), ts.Day())
			return fmt.Sprintf("%v-%v", base, ext)
		}
	}

	cases := map[string]struct {
		ilmCalls []onCall
		imCfg    map[string]interface{}
		cfg      map[string]interface{}
		want     nameFunc
		meta     common.MapStr
	}{
		"without ilm": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"event alias without ilm": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("test"),
			meta: common.MapStr{
				"alias": "test",
			},
		},
		"event index without ilm": {
			ilmCalls: noILM,
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     dateIdx("test"),
			meta: common.MapStr{
				"index": "test",
			},
		},
		"with ilm": {
			ilmCalls: ilmTemplateSettings("test-9.9.9", "test-9.9.9"),
			cfg:      map[string]interface{}{"index": "wrong-%{[agent.version]}"},
			want:     stable("test-9.9.9"),
		},
		"event alias wit ilm": {
			ilmCalls: ilmTemplateSettings("test-9.9.9", "test-9.9.9"),
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     stable("event-alias"),
			meta: common.MapStr{
				"alias": "event-alias",
			},
		},
		"event index with ilm": {
			ilmCalls: ilmTemplateSettings("test-9.9.9", "test-9.9.9"),
			cfg:      map[string]interface{}{"index": "test-%{[agent.version]}"},
			want:     dateIdx("event-index"),
			meta: common.MapStr{
				"index": "event-index",
			},
		},
		"use indices": {
			ilmCalls: ilmTemplateSettings("test-9.9.9", "test-9.9.9"),
			cfg: map[string]interface{}{
				"index": "test-%{[agent.version]}",
				"indices": []map[string]interface{}{
					{"index": "myindex"},
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
			im, err := factory(nil, info, common.MustNewConfigFrom(test.imCfg))
			require.NoError(t, err)

			sel, err := im.BuildSelector(common.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)

			meta := test.meta
			idx, err := sel.Select(&beat.Event{
				Timestamp: ts,
				Fields: common.MapStr{
					"test": "value",
					"agent": common.MapStr{
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
	for name, setup := range map[string]struct {
		tmplEnabled, ilmEnabled, ilmOverwrite bool
		loadTmpl, loadILM                     LoadMode
		ok                                    bool
		warn                                  string
	}{
		"load template with ilm without loading ilm": {
			ilmEnabled: true, tmplEnabled: true, loadILM: LoadModeDisabled,
			warn: "whithout loading ILM policy and alias",
		},
		"load ilm without template": {
			ilmEnabled: true, loadILM: LoadModeUnset,
			warn: "without loading template is not recommended",
		},
		"template disabled but loading enabled": {
			loadTmpl: LoadModeEnabled,
			warn:     "loading not enabled",
		},
		"ilm disabled but loading enabled": {
			loadILM: LoadModeEnabled, tmplEnabled: true,
			warn: "loading not enabled",
		},
		"ilm enabled but loading disabled": {
			ilmEnabled: true, loadILM: LoadModeDisabled,
			warn: "loading not enabled",
		},
		"template enabled but loading disabled": {
			tmplEnabled: true, loadTmpl: LoadModeDisabled,
			warn: "loading not enabled",
		},
		"ilm enabled but overwrite disabled": {
			tmplEnabled: true,
			ilmEnabled:  true, ilmOverwrite: false, loadILM: LoadModeEnabled,
			warn: "Overwriting ILM policy is disabled",
		},
		"everything enabled": {
			tmplEnabled: true,
			ilmEnabled:  true, ilmOverwrite: true,
			ok: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(common.MapStr{
				"setup.ilm.enabled":      setup.ilmEnabled,
				"setup.ilm.overwrite":    setup.ilmOverwrite,
				"setup.template.enabled": setup.tmplEnabled,
			})
			require.NoError(t, err)
			support, err := MakeDefaultSupport(ilm.StdSupport)(nil, beat.Info{}, cfg)
			require.NoError(t, err)
			clientHandler := newMockClientHandler()
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
			c.Settings.Index = (map[string]interface{})(common.MapStr(c.Settings.Index).Clone())
		}
		if c.Settings.Index != nil {
			c.Settings.Source = (map[string]interface{})(common.MapStr(c.Settings.Source).Clone())
		}
		return c
	}

	cfgWith := func(s template.TemplateConfig, mods ...map[string]interface{}) *template.TemplateConfig {
		for _, mod := range mods {
			cfg := common.MustNewConfigFrom(mod)
			s = cloneCfg(s)
			err := cfg.Unpack(&s)
			if err != nil {
				panic(err)
			}
		}
		return &s
	}
	defaultCfg := template.DefaultConfig()

	cases := map[string]struct {
		cfg                   common.MapStr
		loadTemplate, loadILM LoadMode

		err           bool
		tmplCfg       *template.TemplateConfig
		alias, policy string
	}{
		"template default ilm default": {
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9-*",
				"settings.index.lifecycle.name": "test",
				"settings.index.lifecycle.rollover_alias": "test-9.9.9",
			}),
			alias:  "test-9.9.9",
			policy: "test",
		},
		"template default ilm default with alias and policy changed": {
			cfg: common.MapStr{
				"setup.ilm.rollover_alias": "mocktest",
				"setup.ilm.policy_name":    "policy-keep",
			},
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "mocktest",
				"pattern":                       "mocktest-*",
				"settings.index.lifecycle.name": "policy-keep",
				"settings.index.lifecycle.rollover_alias": "mocktest",
			}),
			alias:  "mocktest",
			policy: "policy-keep",
		},
		"template default ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeEnabled,
			tmplCfg:      &defaultCfg,
		},
		"template default loadMode Overwrite ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeOverwrite,
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"overwrite": "true",
			}),
		},
		"template default loadMode Force ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeForce,
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"overwrite": "true",
			}),
		},
		"template loadMode disabled ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeDisabled,
		},
		"template disabled ilm default": {
			cfg: common.MapStr{
				"setup.template.enabled": false,
			},
			alias:  "test-9.9.9",
			policy: "test",
		},
		"template disabled ilm disabled, loadMode Overwrite": {
			cfg: common.MapStr{
				"setup.template.enabled": false,
				"setup.ilm.enabled":      false,
			},
			loadILM: LoadModeOverwrite,
		},
		"template disabled ilm disabled loadMode Force": {
			cfg: common.MapStr{
				"setup.template.enabled": false,
				"setup.ilm.enabled":      false,
			},
			loadILM: LoadModeForce,
			alias:   "test-9.9.9",
			policy:  "test",
		},
		"template loadmode disabled ilm loadMode enabled": {
			loadTemplate: LoadModeDisabled,
			loadILM:      LoadModeEnabled,
			alias:        "test-9.9.9",
			policy:       "test",
		},
		"template default ilm loadMode disabled": {
			loadILM: LoadModeDisabled,
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9-*",
				"settings.index.lifecycle.name": "test",
				"settings.index.lifecycle.rollover_alias": "test-9.9.9",
			}),
		},
		"template loadmode disabled ilm loadmode disabled": {
			loadTemplate: LoadModeDisabled,
			loadILM:      LoadModeDisabled,
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			info := beat.Info{Beat: "test", Version: "9.9.9"}
			factory := MakeDefaultSupport(ilm.StdSupport)
			im, err := factory(nil, info, common.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)

			clientHandler := newMockClientHandler()
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
				assert.Equal(t, test.alias, clientHandler.alias)
				assert.Equal(t, test.policy, clientHandler.policy)
			}
		})
	}
}

func (op mockCreateOp) String() string {
	names := []string{"create-policy", "create-template", "create-alias"}
	if int(op) > len(names) {
		return "unknown"
	}
	return names[op]
}

func newMockClientHandler() *mockClientHandler {
	return &mockClientHandler{}
}

func (h *mockClientHandler) Load(config template.TemplateConfig, _ beat.Info, fields []byte, migration bool) error {
	h.recordOp(mockCreateTemplate)
	h.tmplForce = config.Overwrite
	h.tmplCfg = &config
	return nil
}

func (h *mockClientHandler) CheckILMEnabled(m ilm.Mode) (bool, error) {
	return m == ilm.ModeEnabled || m == ilm.ModeAuto, nil
}

func (h *mockClientHandler) HasAlias(name string) (bool, error) {
	return h.alias == name, nil
}

func (h *mockClientHandler) CreateAlias(alias ilm.Alias) error {
	h.recordOp(mockCreateAlias)
	h.alias = alias.Name
	return nil
}

func (h *mockClientHandler) HasILMPolicy(name string) (bool, error) {
	return h.policy == name, nil
}

func (h *mockClientHandler) CreateILMPolicy(policy ilm.Policy) error {
	h.recordOp(mockCreatePolicy)
	h.policy = policy.Name
	return nil
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
