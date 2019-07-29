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

package ilm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestDefaultSupport_Init(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}

	t.Run("mode from config", func(t *testing.T) {
		cases := map[string]Mode{
			"true":  ModeEnabled,
			"false": ModeDisabled,
			"auto":  ModeAuto,
		}
		for setting, expected := range cases {
			expected := expected
			t.Run(setting, func(t *testing.T) {
				cfg := common.MustNewConfigFrom(map[string]interface{}{
					"enabled":        setting,
					"rollover_alias": "test",
				})

				s, err := DefaultSupport(nil, info, cfg)
				require.NoError(t, err)
				assert.Equal(t, expected, s.Mode())
			})
		}
	})

	t.Run("with an empty rollover_alias", func(t *testing.T) {
		_, err := DefaultSupport(nil, info, common.MustNewConfigFrom(
			map[string]interface{}{
				"enabled":        true,
				"rollover_alias": "",
				"pattern":        "01",
				"check_exists":   false,
				"overwrite":      true,
			},
		))
		require.Error(t, err)
	})

	t.Run("with custom config", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, common.MustNewConfigFrom(
			map[string]interface{}{
				"enabled":        true,
				"name":           "test-%{[agent.version]}",
				"rollover_alias": "alias",
				"pattern":        "01",
				"check_exists":   false,
				"overwrite":      true,
			},
		))
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.Equal(true, s.overwrite)
		assert.Equal(false, s.checkExists)
		assert.Equal(ModeEnabled, s.Mode())
		assert.Equal(DefaultPolicy, common.MapStr(s.Policy().Body))
		assert.Equal(Alias{Name: "alias", Pattern: "01"}, s.Alias())
	})

	t.Run("with custom alias config with fieldref", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, common.MustNewConfigFrom(
			map[string]interface{}{
				"enabled":        true,
				"rollover_alias": "alias-%{[agent.version]}",
				"pattern":        "01",
				"check_exists":   false,
				"overwrite":      true,
			},
		))
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.Equal(true, s.overwrite)
		assert.Equal(false, s.checkExists)
		assert.Equal(ModeEnabled, s.Mode())
		assert.Equal(DefaultPolicy, common.MapStr(s.Policy().Body))
		assert.Equal(Alias{Name: "alias-9.9.9", Pattern: "01"}, s.Alias())
	})

	t.Run("with default alias", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, common.MustNewConfigFrom(
			map[string]interface{}{
				"enabled":      true,
				"pattern":      "01",
				"check_exists": false,
				"overwrite":    true,
			},
		))
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.Equal(true, s.overwrite)
		assert.Equal(false, s.checkExists)
		assert.Equal(ModeEnabled, s.Mode())
		assert.Equal(DefaultPolicy, common.MapStr(s.Policy().Body))
		assert.Equal(Alias{Name: "test-9.9.9", Pattern: "01"}, s.Alias())
	})

	t.Run("load external policy", func(t *testing.T) {
		s, err := DefaultSupport(nil, info, common.MustNewConfigFrom(
			common.MapStr{"policy_file": "testfiles/custom.json"},
		))
		require.NoError(t, err)
		assert.Equal(t, common.MapStr{"hello": "world"}, s.Policy().Body)
	})
}

func TestDefaultSupport_Manager_Enabled(t *testing.T) {
	cases := map[string]struct {
		calls []onCall
		cfg   map[string]interface{}
		b     bool
		fail  error
		err   bool
	}{
		"disabled via config": {
			cfg: map[string]interface{}{"enabled": false},
		},
		"disabled via handler": {
			calls: []onCall{
				onCheckILMEnabled(ModeAuto).Return(false, nil),
			},
		},
		"enabled via handler": {
			calls: []onCall{
				onCheckILMEnabled(ModeAuto).Return(true, nil),
			},
			b: true,
		},
		"handler confirms enabled flag": {
			calls: []onCall{
				onCheckILMEnabled(ModeEnabled).Return(true, nil),
			},
			cfg: map[string]interface{}{"enabled": true},
			b:   true,
		},
		"fail enabled": {
			calls: []onCall{
				onCheckILMEnabled(ModeEnabled).Return(false, nil),
			},
			cfg:  map[string]interface{}{"enabled": true},
			fail: ErrESILMDisabled,
		},
		"io error": {
			calls: []onCall{
				onCheckILMEnabled(ModeAuto).Return(false, errors.New("ups")),
			},
			cfg: map[string]interface{}{},
			err: true,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := test.cfg
			if cfg == nil {
				cfg = map[string]interface{}{}
			}

			h := newMockHandler(test.calls...)
			m := createManager(t, h, test.cfg)
			b, err := m.Enabled()

			if test.fail == nil && !test.err {
				require.NoError(t, err)
			}
			if test.err || test.fail != nil {
				require.Error(t, err)
			}
			if test.fail != nil {
				assert.Equal(t, test.fail, ErrReason(err))
			}

			assert.Equal(t, test.b, b)
			h.AssertExpectations(t)
		})
	}
}

func TestDefaultSupport_Manager_EnsureAlias(t *testing.T) {
	alias := Alias{
		Name:    "test-9.9.9",
		Pattern: ilmDefaultPattern,
	}

	cases := map[string]struct {
		calls []onCall
		cfg   map[string]interface{}
		fail  error
	}{
		"create new alias": {
			calls: []onCall{
				onHasAlias(alias.Name).Return(false, nil),
				onCreateAlias(alias).Return(nil),
			},
		},
		"alias already exists": {
			calls: []onCall{
				onHasAlias(alias.Name).Return(true, nil),
			},
		},
		"fail": {
			calls: []onCall{
				onHasAlias(alias.Name).Return(false, nil),
				onCreateAlias(alias).Return(errOf(ErrRequestFailed)),
			},
			fail: ErrRequestFailed,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := test.cfg
			if cfg == nil {
				cfg = map[string]interface{}{"alias": "test"}
			}

			h := newMockHandler(test.calls...)
			m := createManager(t, h, test.cfg)
			err := m.EnsureAlias()

			if test.fail == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, test.fail, ErrReason(err))
			}
			h.AssertExpectations(t)
		})
	}
}

func TestDefaultSupport_Manager_EnsurePolicy(t *testing.T) {
	testPolicy := Policy{
		Name: "test-9.9.9",
		Body: DefaultPolicy,
	}

	cases := map[string]struct {
		calls     []onCall
		overwrite bool
		cfg       map[string]interface{}
		create    bool
		fail      error
	}{
		"create new policy": {
			create: true,
			calls: []onCall{
				onHasILMPolicy(testPolicy.Name).Return(false, nil),
				onCreateILMPolicy(testPolicy).Return(nil),
			},
		},
		"policy already exists": {
			calls: []onCall{
				onHasILMPolicy(testPolicy.Name).Return(true, nil),
			},
		},
		"overwrite existing": {
			overwrite: true,
			calls: []onCall{
				onCreateILMPolicy(testPolicy).Return(nil),
			},
		},
		"fail": {
			calls: []onCall{
				onHasILMPolicy(testPolicy.Name).Return(false, nil),
				onCreateILMPolicy(testPolicy).Return(errOf(ErrRequestFailed)),
			},
			fail: ErrRequestFailed,
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := test.cfg
			if cfg == nil {
				cfg = map[string]interface{}{"name": "test"}
			}

			h := newMockHandler(test.calls...)
			m := createManager(t, h, test.cfg)
			created, err := m.EnsurePolicy(test.overwrite)

			if test.fail == nil {
				assert.Equal(t, test.create, created)
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, test.fail, ErrReason(err))
			}

			h.AssertExpectations(t)
		})
	}
}

func createManager(t *testing.T, h ClientHandler, cfg map[string]interface{}) Manager {
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	s, err := DefaultSupport(nil, info, common.MustNewConfigFrom(cfg))
	require.NoError(t, err)
	return s.Manager(h)
}
