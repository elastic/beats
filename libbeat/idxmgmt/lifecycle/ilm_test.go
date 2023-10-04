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

package lifecycle

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestDefaultSupport_Init(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}

	t.Run("with custom config", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, true)
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.True(s.Enabled())
	})

}

func TestDefaultSupport_Manager_Enabled_Serverless(t *testing.T) {
	cfg := DefaultDSLConfig(beat.Info{Name: "test"})
	runEnabledTests(t, cfg)
}

func TestDefaultSupport_Manager_Enabled(t *testing.T) {
	cfg := DefaultILMConfig(beat.Info{Name: "test"})
	runEnabledTests(t, cfg)
}

func runEnabledTests(t *testing.T, cfg LifecycleConfig) {
	cases := map[string]struct {
		calls         []onCall
		cfg           LifecycleConfig
		expectEnabled bool
		isEnabled     bool
		fail          error
		err           bool
	}{
		"disabled via config": {
			cfg:           LifecycleConfig{ILM: Config{Enabled: false}, DSL: Config{Enabled: false}},
			expectEnabled: false,
			isEnabled:     false,
		},
		"disabled via handler": {
			calls: []onCall{
				onCheckEnabled().Return(false, ErrESILMDisabled),
			},
			expectEnabled: false,
			isEnabled:     true,
			cfg:           cfg,
			err:           true,
		},
		"enabled via handler": {
			calls: []onCall{
				onCheckEnabled().Return(true, nil),
			},
			expectEnabled: true,
			isEnabled:     true,
			cfg:           cfg,
		},
		"handler confirms enabled flag": {
			calls: []onCall{
				onCheckEnabled().Return(true, nil),
			},
			cfg:           cfg,
			expectEnabled: true,
			isEnabled:     true,
		},
		"io error": {
			calls: []onCall{
				onCheckEnabled().Return(false, errors.New("ups")),
			},
			cfg:           cfg,
			expectEnabled: false,
			isEnabled:     true,
			err:           true,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {

			testHandler := newMockHandler(test.cfg, Policy{}, test.calls...)
			testManager := createManager(t, testHandler, test.isEnabled)
			enabled, err := testManager.CheckEnabled()

			if test.fail == nil && !test.err {
				assert.NoError(t, err)
			}
			if test.err || test.fail != nil {
				assert.Error(t, err)
			}
			if test.fail != nil {
				assert.Equal(t, test.fail, err)
			}

			assert.Equal(t, test.expectEnabled, enabled)
			testHandler.AssertExpectations(t)
		})
	}
}

func TestDefaultSupport_Manager_EnsurePolicy_Serverless(t *testing.T) {
	testPolicy := Policy{
		Name: "test",
		Body: DefaultDSLPolicy,
	}
	cfg := DefaultDSLConfig(beat.Info{Name: "test"})
	runEnsurePolicyTest(t, testPolicy, cfg)
}

func TestDefaultSupport_Manager_EnsurePolicy(t *testing.T) {
	testPolicy := Policy{
		Name: "test",
		Body: DefaultILMPolicy,
	}
	cfg := DefaultILMConfig(beat.Info{Name: "test"})
	runEnsurePolicyTest(t, testPolicy, cfg)
}

func runEnsurePolicyTest(t *testing.T, testPolicy Policy, cfg LifecycleConfig) {
	cases := map[string]struct {
		calls     []onCall
		overwrite bool
		cfg       LifecycleConfig
		enabled   bool
		create    bool
		fail      error
	}{
		"create new policy": {
			create: true,
			calls: []onCall{
				onCheckExists().Return(true),
				onHasPolicy().Return(false, nil),
				onCreatePolicyFromConfig().Return(nil),
			},
			cfg:     cfg,
			enabled: true,
		},
		"policy already exists": {
			create: false,
			calls: []onCall{
				onCheckExists().Return(true),
				onHasPolicy().Return(true, nil),
			},
			cfg:     cfg,
			enabled: true,
		},
		"overwrite": {
			overwrite: true,
			create:    true,
			enabled:   true,
			cfg:       cfg,
			calls: []onCall{
				onCheckExists().Return(true),
				onCreatePolicyFromConfig().Return(nil),
			},
		},
		"fail": {
			calls: []onCall{
				onCheckExists().Return(true),
				onHasPolicy().Return(false, nil),
				onCreatePolicyFromConfig().Return(ErrRequestFailed),
			},
			fail:    ErrRequestFailed,
			cfg:     cfg,
			enabled: true,
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			h := newMockHandler(test.cfg, testPolicy, test.calls...)
			m := createManager(t, h, test.enabled)
			created, err := m.EnsurePolicy(test.overwrite)

			if test.fail == nil {
				assert.Equal(t, test.create, created)
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, test.fail, err)
			}

			h.AssertExpectations(t)
		})
	}
}

func createManager(t *testing.T, h ClientHandler, enabled bool) Manager {
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	s, err := DefaultSupport(nil, info, enabled)
	require.NoError(t, err)
	return s.Manager(h)
}
