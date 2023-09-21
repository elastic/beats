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
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
)

func TestDefaultSupport_Init(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}

	t.Run("with custom config", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, LifecycleConfig{
			ILM: Config{
				Enabled:     true,
				PolicyName:  *fmtstr.MustCompileEvent("test-%{[agent.version]}"),
				CheckExists: false,
				Overwrite:   true,
			},
		})
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.Equal(true, s.cfg.ILM.Overwrite)
		assert.Equal(false, s.cfg.ILM.CheckExists)
		assert.Equal(true, s.Enabled())
	})

	t.Run("with custom alias config with fieldref", func(t *testing.T) {
		tmp, err := DefaultSupport(nil, info, LifecycleConfig{
			ILM: Config{
				Enabled:     true,
				CheckExists: false,
				Overwrite:   true,
			},
		})
		require.NoError(t, err)

		s := tmp.(*stdSupport)
		assert := assert.New(t)
		assert.Equal(true, s.cfg.ILM.Overwrite)
		assert.Equal(false, s.cfg.ILM.CheckExists)
		assert.Equal(true, s.Enabled())
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
		calls   []onCall
		cfg     LifecycleConfig
		enabled bool
		fail    error
		err     bool
	}{
		"disabled via config": {
			cfg: LifecycleConfig{ILM: Config{Enabled: false}, DSL: Config{Enabled: false}},
		},
		"disabled via handler": {
			calls: []onCall{
				onCheckEnabled().Return(false, ErrESILMDisabled),
			},
			cfg: cfg,
			err: true,
		},
		"enabled via handler": {
			calls: []onCall{
				onCheckEnabled().Return(true, nil),
			},
			enabled: true,
			cfg:     cfg,
		},
		"handler confirms enabled flag": {
			calls: []onCall{
				onCheckEnabled().Return(true, nil),
			},
			cfg:     cfg,
			enabled: true,
		},
		"io error": {
			calls: []onCall{
				onCheckEnabled().Return(false, errors.New("ups")),
			},
			cfg: cfg,
			err: true,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {

			testHandler := newMockHandler(test.cfg, Policy{}, test.calls...)
			testManager := createManager(t, testHandler, test.cfg)
			enabled, err := testManager.CheckEnabled()

			if test.fail == nil && !test.err {
				require.NoError(t, err)
			}
			if test.err || test.fail != nil {
				require.Error(t, err)
			}
			if test.fail != nil {
				assert.Equal(t, test.fail, err)
			}

			assert.Equal(t, test.enabled, enabled)
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
			cfg: cfg,
		},
		"policy already exists": {
			create: false,
			calls: []onCall{
				onCheckExists().Return(true),
				onHasPolicy().Return(true, nil),
			},
			cfg: cfg,
		},
		"overwrite": {
			overwrite: true,
			create:    true,
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
				onCreatePolicyFromConfig().Return(errOf(ErrRequestFailed)),
			},
			fail: ErrRequestFailed,
			cfg:  cfg,
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			h := newMockHandler(test.cfg, testPolicy, test.calls...)
			m := createManager(t, h, test.cfg)
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

func createManager(t *testing.T, h ClientHandler, cfg LifecycleConfig) Manager {
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	s, err := DefaultSupport(nil, info, cfg)
	require.NoError(t, err)
	return s.Manager(h)
}
