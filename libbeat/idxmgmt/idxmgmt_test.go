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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
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

func TestDefaultSupport_TemplateConfig(t *testing.T) {
	ilmTemplateSettings := func(s ilm.TemplateSettings) []onCall {
		return []onCall{
			onMode().Return(ilm.ModeEnabled),
			onTemplate().Return(s),
		}
	}

	cloneCfg := func(c template.TemplateConfig) template.TemplateConfig {
		if c.AppendFields != nil {
			tmp := make(common.Fields, len(c.AppendFields))
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

	cfgWith := func(s template.TemplateConfig, mods ...map[string]interface{}) template.TemplateConfig {
		for _, mod := range mods {
			cfg := common.MustNewConfigFrom(mod)
			s = cloneCfg(s)
			err := cfg.Unpack(&s)
			if err != nil {
				panic(err)
			}
		}
		return s
	}

	cases := map[string]struct {
		ilmCalls []onCall
		cfg      map[string]interface{}
		want     template.TemplateConfig
		fail     bool
	}{
		"default template config": {
			want: template.DefaultConfig,
		},
		"default template with ilm": {
			ilmCalls: ilmTemplateSettings(ilm.TemplateSettings{
				Alias:      "alias",
				Pattern:    "alias-*",
				PolicyName: "test-9.9.9",
			}),
			want: cfgWith(template.DefaultConfig, map[string]interface{}{
				"name":                          "alias",
				"pattern":                       "alias-*",
				"settings.index.lifecycle.name": "test-9.9.9",
				"settings.index.lifecycle.rollover_alias": "alias",
			}),
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			info := beat.Info{Beat: "test", Version: "9.9.9"}
			factory := MakeDefaultSupport(makeMockILMSupport(test.ilmCalls...))
			im, err := factory(nil, info, common.MustNewConfigFrom(test.cfg))
			require.NoError(t, err)
			withILM := len(test.ilmCalls) > 0

			tmpl, err := im.TemplateConfig(withILM)
			if test.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, tmpl)
			}
		})
	}
}
