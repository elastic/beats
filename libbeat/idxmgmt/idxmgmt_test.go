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

func TestDefaultSupport_Enabled(t *testing.T) {
	cases := map[string]struct {
		ilmCalls []onCall
		cfg      map[string]interface{}
		want     bool
	}{
		"templates and ilm disabled": {
			want: false,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeDisabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
		"templates only": {
			want: true,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeDisabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": true,
			},
		},
		"ilm only": {
			want: true,
			ilmCalls: []onCall{
				onMode().Return(ilm.ModeEnabled),
			},
			cfg: map[string]interface{}{
				"setup.template.enabled": false,
			},
		},
		"ilm tentatively": {
			want: true,
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
			assert.Equal(t, test.want, im.Enabled())
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

func TestDefaultSupport_TemplateHandling(t *testing.T) {
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
		"template default, ilm default": {
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"overwrite":                     "true",
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9-*",
				"settings.index.lifecycle.name": "test-9.9.9",
				"settings.index.lifecycle.rollover_alias": "test-9.9.9",
			}),
			alias:  "test-9.9.9",
			policy: "test-9.9.9",
		},
		"template default, ilm default with alias and policy changed": {
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
		"template default, ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeEnabled,
			tmplCfg:      &defaultCfg,
		},
		"template loadMode disabled, ilm disabled": {
			cfg: common.MapStr{
				"setup.ilm.enabled": false,
			},
			loadTemplate: LoadModeDisabled,
		},
		"template disabled, ilm default": {
			cfg: common.MapStr{
				"setup.template.enabled": false,
			},
			alias:  "test-9.9.9",
			policy: "test-9.9.9",
		},
		"template loadmode disabled, ilm loadMode enabled": {
			loadTemplate: LoadModeDisabled,
			loadILM:      LoadModeEnabled,
			alias:        "test-9.9.9",
			policy:       "test-9.9.9",
		},
		"template default, ilm loadMode disabled": {
			loadILM: LoadModeDisabled,
			tmplCfg: cfgWith(template.DefaultConfig(), map[string]interface{}{
				"name":                          "test-9.9.9",
				"pattern":                       "test-9.9.9-*",
				"settings.index.lifecycle.name": "test-9.9.9",
				"settings.index.lifecycle.rollover_alias": "test-9.9.9",
			}),
		},
		"template loadmode disabled, ilm loadmode disabled": {
			loadTemplate: LoadModeDisabled,
			loadILM:      LoadModeDisabled,
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			info := beat.Info{Beat: "test", Version: "9.9.9"}
			factory := MakeDefaultSupport(nil)
			im, err := factory(nil, info, common.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)

			clientHandler := newMockClientHandler()
			manager := im.Manager(clientHandler, BeatsAssets([]byte("testbeat fields")))
			err = manager.Setup(test.loadTemplate, test.loadILM)
			if test.err {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if test.tmplCfg == nil {
					assert.Nil(t, clientHandler.tl.tmplCfg)
				} else {
					assert.Equal(t, test.tmplCfg, clientHandler.tl.tmplCfg)
				}
				assert.Equal(t, test.alias, clientHandler.il.alias)
				assert.Equal(t, test.policy, clientHandler.il.policy)
			}
		})
	}
}

func newMockClientHandler() *mockClientHandler {
	tl := mockTemplateLoader{}
	il := mockILMClientHandler{}
	return &mockClientHandler{&il, &tl, &tl, &il}
}

type mockClientHandler struct {
	ilm.ClientHandler
	template.Loader

	tl *mockTemplateLoader
	il *mockILMClientHandler
}

type mockTemplateLoader struct {
	tmplCfg *template.TemplateConfig
	force   bool
}

func (l *mockTemplateLoader) Load(config template.TemplateConfig, _ beat.Info, fields []byte, migration bool) error {
	l.force = config.Overwrite
	l.tmplCfg = &config
	return nil
}

type mockILMClientHandler struct {
	alias, policy string
}

func (ch *mockILMClientHandler) CheckILMEnabled(m ilm.Mode) (bool, error) {
	return m == ilm.ModeEnabled || m == ilm.ModeAuto, nil
}

func (ch *mockILMClientHandler) HasAlias(name string) (bool, error) {
	return ch.alias == name, nil
}

func (ch *mockILMClientHandler) CreateAlias(alias ilm.Alias) error {
	ch.alias = alias.Name
	return nil
}

func (ch *mockILMClientHandler) HasILMPolicy(name string) (bool, error) {
	return ch.policy == name, nil
}

func (ch *mockILMClientHandler) CreateILMPolicy(policy ilm.Policy) error {
	ch.policy = policy.Name
	return nil
}
