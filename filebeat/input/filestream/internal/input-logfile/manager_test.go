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

package input_logfile

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const testPluginName = "my_test_plugin"

type testSource struct {
	name string
}

func (s *testSource) Name() string {
	return s.name
}

type noopProspector struct{}

func (m noopProspector) Init(_, _ ProspectorCleaner, _ func(Source) string) error {
	return nil
}

func (m noopProspector) Run(_ v2.Context, _ StateMetadataUpdater, _ HarvesterGroup) {}

func (m noopProspector) Test() error {
	return nil
}

func TestSourceIdentifier_ID(t *testing.T) {
	testCases := map[string]struct {
		userID            string
		sources           []*testSource
		expectedSourceIDs []string
	}{
		"plugin with no user configured ID": {
			sources: []*testSource{
				{"unique_name"},
				{"another_unique_name"},
			},
			expectedSourceIDs: []string{
				testPluginName + "::.global::unique_name",
				testPluginName + "::.global::another_unique_name",
			},
		},
	}

	for name, test := range testCases {
		test := test

		t.Run(name, func(t *testing.T) {
			srcIdentifier, err := newSourceIdentifier(testPluginName, test.userID)
			if err != nil {
				t.Fatalf("cannot create identifier: %v", err)
			}

			for i, src := range test.sources {
				t.Run(name+"_with_src: "+src.Name(), func(t *testing.T) {
					srcID := srcIdentifier.ID(src)
					assert.Equal(t, test.expectedSourceIDs[i], srcID)
				})
			}
		})
	}
}

func TestSourceIdentifier_MatchesInput(t *testing.T) {
	testCases := map[string]struct {
		userID      string
		matchingIDs []string
	}{
		"plugin with no user configured ID": {
			matchingIDs: []string{
				testPluginName + "::.global::my_id",
				testPluginName + "::.global::path::my_id",
				testPluginName + "::.global::" + testPluginName + "::my_id",
			},
		},
		"plugin with user configured ID": {
			userID: "my-id",
			matchingIDs: []string{
				testPluginName + "::my-id::my_id",
				testPluginName + "::my-id::path::my_id",
				testPluginName + "::my-id::" + testPluginName + "::my_id",
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			srcIdentifier, err := newSourceIdentifier(testPluginName, test.userID)
			if err != nil {
				t.Fatalf("cannot create identifier: %v", err)
			}

			for _, id := range test.matchingIDs {
				t.Run(name+"_with_id: "+id, func(t *testing.T) {
					assert.True(t, srcIdentifier.MatchesInput(id))
				})
			}
		})
	}
}

func TestSourceIdentifier_NotMatchesInput(t *testing.T) {
	testCases := map[string]struct {
		userID         string
		notMatchingIDs []string
	}{
		"plugin with user configured ID": {
			userID: "my-id",
			notMatchingIDs: []string{
				testPluginName + "-other-id::my_id",
				"my-id::path::my_id",
			},
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			srcIdentifier, err := newSourceIdentifier(testPluginName, test.userID)
			if err != nil {
				t.Fatalf("cannot create identifier: %v", err)
			}

			for _, id := range test.notMatchingIDs {
				t.Run(name+"_with_id: "+id, func(t *testing.T) {
					assert.False(t, srcIdentifier.MatchesInput(id))
				})
			}
		})
	}
}

func TestSourceIdentifierNoAccidentalMatches(t *testing.T) {
	noIDIdentifier, err := newSourceIdentifier(testPluginName, "")
	if err != nil {
		t.Fatalf("cannot create identifier: %v", err)
	}
	withIDIdentifier, err := newSourceIdentifier(testPluginName, "id")
	if err != nil {
		t.Fatalf("cannot create identifier: %v", err)
	}

	src := &testSource{"test"}
	assert.NotEqual(t, noIDIdentifier.ID(src), withIDIdentifier.ID(src))
	assert.False(t, noIDIdentifier.MatchesInput(withIDIdentifier.ID(src)))
	assert.False(t, withIDIdentifier.MatchesInput(noIDIdentifier.ID(src)))
}

func TestInputManager_Create(t *testing.T) {
	t.Run("Checking config does not print duplicated id warning",
		func(t *testing.T) {
			storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
			testStore, err := storeReg.Get("test")
			require.NoError(t, err)

			log, buff := newBufferLogger()

			cim := &InputManager{
				Logger:     log,
				StateStore: testStateStore{Store: testStore},
				Configure: func(_ *config.C) (Prospector, Harvester, error) {
					return nil, nil, nil
				}}
			cfg, err := config.NewConfigFrom("id: my-id")
			require.NoError(t, err)

			_, err = cim.Create(cfg)
			require.ErrorIs(t, err, errNoInputRunner)
			err = cim.Delete(cfg)
			require.NoError(t, err)

			// Create again to ensure now warning regarding duplicated ID will
			// be logged.
			_, err = cim.Create(cfg)
			require.ErrorIs(t, err, errNoInputRunner)
			err = cim.Delete(cfg)
			require.NoError(t, err)

			assert.NotContains(t, buff.String(),
				"filestream input with ID")
			assert.NotContains(t, buff.String(),
				"already exists")
		})

	t.Run("failed input has its ID removed from the IDs list", func(t *testing.T) {
		storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
		testStore, err := storeReg.Get("test")
		require.NoError(t, err)

		log, _ := newBufferLogger()

		cim := &InputManager{
			Logger:     log,
			StateStore: testStateStore{Store: testStore},
			Configure: func(cfg *config.C) (Prospector, Harvester, error) {
				var wg sync.WaitGroup

				settings := struct {
					ID    string   `config:"id"`
					Paths []string `config:"paths"`
				}{}

				if err := cfg.Unpack(&settings); err != nil {
					return nil, nil, err
				}

				for _, path := range settings.Paths {
					if strings.Contains(path, "**/**") {
						return nil, nil, errors.New("double ** is not allowed in a glob")
					}
				}

				return &noopProspector{}, &mockHarvester{onRun: correctOnRun, wg: &wg}, nil
			}}
		invalidCfg := config.MustNewConfigFrom(`
type: filestream
id: t-wing
paths:
  - "/**/**/foo" # double ** is invalid for Filestream
`)

		// Create a valid config with the same ID
		validCfg := config.MustNewConfigFrom(`
type: filestream
id: t-wing
paths:
  - /var/log/bar
`)

		// Happy path, if an input fails to start, it's ID is removed from cim.ids list and
		// can be re-used.
		t.Run("happy path", func(t *testing.T) {
			// Attempt to create the first input with the invalid configuration
			_, err = cim.Create(invalidCfg)
			require.Error(
				t,
				err,
				"'/**/**' is not supported, input creation must fail")

			require.Len(t, cim.ids, 0, "no ID must be present in cim.ids")
			// Attempt to create the second input with the valid configuration
			_, err = cim.Create(validCfg)
			require.NoError(
				t,
				err,
				"The same ID can be re-used after an input fails to start")
			require.EqualValues(
				t,
				map[string]struct{}{"t-wing": {}},
				cim.ids,
				"only 't-wing' must be present in cim.ids")

			// "Stop" the input to have the manager in a consistent state
			// using the same flow as an actually running input would
			cim.StopInput("t-wing")
		})

		// Failure scenario: an input with ID X is already running, then an invalid
		// input configuration with the same ID is used while
		// 'allow_deprecated_id_duplication: true'. In this case, the invalid input
		// must fail to start, but the ID from the valid one must stay in cim.ids
		t.Run("running input with the same ID as invalid one", func(t *testing.T) {
			// Attempt to create the input with the valid configuration
			_, err = cim.Create(validCfg)
			require.NoError(t, err, "This input must start")

			require.NoError(t,
				invalidCfg.SetBool("allow_deprecated_id_duplication", -1, true),
				"setting config must not fail")

			// Attempt to create an invalid input with the same ID
			_, err = cim.Create(invalidCfg)
			require.Error(t, err, "'/**/**' is not supported, input creation must fail")
			// The ID of the valid input must still be in the ids list
			require.EqualValues(
				t,
				map[string]struct{}{"t-wing": {}},
				cim.ids,
				"only 't-wing' must be present in cim.ids")
		})
	})
}

func newBufferLogger() (*logp.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	writeSyncer := zapcore.AddSync(buf)
	log := logp.NewLogger("", zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
		return zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	}))
	return log, buf
}
