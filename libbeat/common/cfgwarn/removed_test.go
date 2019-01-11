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

package cfgwarn

import (
	"errors"
	"testing"

	"github.com/joeshaw/multierror"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestRemovedSetting(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *common.Config
		lookup   string
		expected error
	}{
		{
			name:   "no obsolete setting",
			lookup: "notfound",
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"hello.world": "ok",
			}),
			expected: nil,
		},
		{
			name:   "obsolete setting found",
			lookup: "hello",
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"hello.world": "ok",
			}),
			expected: errors.New("setting 'hello' has been removed"),
		},
	}

	functions := []struct {
		name string
		fn   func(*common.Config, string) error
	}{
		{name: "checkRemovedSetting", fn: checkRemovedSetting},
		{name: "checkRemoved6xSetting", fn: CheckRemoved6xSetting},
	}

	for _, function := range functions {
		t.Run(function.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					err := function.fn(test.cfg, test.lookup)
					assert.Equal(t, test.expected, err)
				})
			}
		})
	}
}

func TestRemovedSettings(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *common.Config
		lookup   []string
		expected error
	}{
		{
			name:   "no obsolete setting",
			lookup: []string{"notfound"},
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"hello.world": "ok",
			}),
			expected: nil,
		},
		{
			name:   "obsolete setting found",
			lookup: []string{"hello"},
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"hello.world": "ok",
			}),
			expected: multierror.Errors{errors.New("setting 'hello' has been removed")}.Err(),
		},
		{
			name:   "multiple obsolete settings",
			lookup: []string{"hello", "bad"},
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"hello.world": "ok",
				"bad":         "true",
			}),
			expected: multierror.Errors{
				errors.New("setting 'hello' has been removed"),
				errors.New("setting 'bad' has been removed"),
			}.Err(),
		},
		{
			name:   "multiple obsolete settings not on first level",
			lookup: []string{"filebeat.config.prospectors", "filebeat.prospectors"},
			cfg: common.MustNewConfigFrom(map[string]interface{}{
				"filebeat.prospectors":        "ok",
				"filebeat.config.prospectors": map[string]interface{}{"ok": "ok1"},
			}),
			expected: multierror.Errors{
				errors.New("setting 'filebeat.config.prospectors' has been removed"),
				errors.New("setting 'filebeat.prospectors' has been removed"),
			}.Err(),
		},
	}

	functions := []struct {
		name string
		fn   func(*common.Config, ...string) error
	}{
		{name: "checkRemovedSetting", fn: checkRemovedSettings},
		{name: "checkRemoved6xSetting", fn: CheckRemoved6xSettings},
	}

	for _, function := range functions {
		t.Run(function.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					err := checkRemovedSettings(test.cfg, test.lookup...)
					assert.Equal(t, test.expected, err)
				})
			}
		})
	}
}
