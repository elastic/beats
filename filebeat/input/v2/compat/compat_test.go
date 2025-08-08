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

package compat

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/internal/inputest"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestRunnerFactory_CheckConfig(t *testing.T) {
	t.Run("does not run or test configured input", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		var countConfigure, countTest, countRun int

		// setup
		plugins := inputest.SinglePlugin("test", &inputest.MockInputManager{
			OnConfigure: func(_ *conf.C) (v2.Input, error) {
				countConfigure++
				return &inputest.MockInput{
					OnTest: func(_ v2.TestContext) error { countTest++; return nil },
					OnRun:  func(_ v2.Context, _ beat.PipelineConnector) error { countRun++; return nil },
				}, nil
			},
		})
		loader := inputest.MustNewTestLoader(t, plugins, "type", "test")
		factory := RunnerFactory(log, beat.Info{}, monitoring.NewRegistry(), loader.Loader)

		// run
		err := factory.CheckConfig(conf.NewConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// validate: configured an input, but do not run test or run
		assert.Equal(t, 1, countConfigure)
		assert.Equal(t, 0, countTest)
		assert.Equal(t, 0, countRun)
	})

	t.Run("does not cause input ID duplication", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		var countConfigure, countTest, countRun int
		var runWG sync.WaitGroup
		var ids = map[string]int{}
		var idsMu sync.Mutex

		// setup
		plugins := inputest.SinglePlugin("test", &inputest.MockInputManager{
			OnConfigure: func(cfg *conf.C) (v2.Input, error) {
				idsMu.Lock()
				defer idsMu.Unlock()
				id, err := cfg.String("id", -1)
				assert.NoError(t, err, "OnConfigure: could not get 'id' fom config")
				idsCount := ids[id]
				ids[id] = idsCount + 1

				countConfigure++
				return &inputest.MockInput{
					OnTest: func(_ v2.TestContext) error { countTest++; return nil },
					OnRun: func(_ v2.Context, _ beat.PipelineConnector) error {
						countRun++
						runWG.Done()
						return nil
					},
				}, nil
			},
		})
		loader := inputest.MustNewTestLoader(t, plugins, "type", "test")
		factory := RunnerFactory(
			log,
			beat.Info{Logger: log},
			monitoring.NewRegistry(),
			loader.Loader)

		inputID := "filestream-kubernetes-pod-aee2af1c6365ecdd72416f44aab49cd8bdc7522ab008c39784b7fd9d46f794a4"
		inputCfg := fmt.Sprintf(`
id: %s
parsers:
  - container: null
paths:
  - /var/log/containers/*aee2af1c6365ecdd72416f44aab49cd8bdc7522ab008c39784b7fd9d46f794a4.log
prospector:
  scanner:
    symlinks: true
type: test
`, inputID)

		runner, err := factory.Create(nil, conf.MustNewConfigFrom(inputCfg))
		require.NoError(t, err, "could not create input")

		runWG.Add(1)
		runner.Start()
		defer runner.Stop()
		// wait input to be running
		runWG.Wait()

		err = factory.CheckConfig(conf.MustNewConfigFrom(inputCfg))
		require.NoError(t, err, "unexpected error when calling CheckConfig")

		// validate: configured an input, but do not run test or run
		assert.Equal(t, 2, countConfigure, "OnConfigure should be called only 2 times")
		assert.Equal(t, 0, countTest, "OnTest should not have been called")
		assert.Equal(t, 1, countRun, "OnRun should be called only once")
		idsMu.Lock()
		assert.Equal(t, 1, ids[inputID])
		idsMu.Unlock()
	})

	t.Run("fail if input type is unknown to loader", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		plugins := inputest.SinglePlugin("test", inputest.ConstInputManager(nil))
		loader := inputest.MustNewTestLoader(t, plugins, "type", "")
		factory := RunnerFactory(
			log,
			beat.Info{Logger: log},
			monitoring.NewRegistry(),
			loader.Loader)

		// run
		err := factory.CheckConfig(conf.MustNewConfigFrom(map[string]interface{}{
			"type": "unknown",
		}))
		assert.Error(t, err)
	})
}

func TestRunnerFactory_CreateAndRun(t *testing.T) {
	t.Run("runner can correctly start and stop inputs", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		var countRun int
		var wg sync.WaitGroup
		plugins := inputest.SinglePlugin("test", inputest.ConstInputManager(&inputest.MockInput{
			OnRun: func(ctx v2.Context, _ beat.PipelineConnector) error {
				defer wg.Done()
				countRun++
				<-ctx.Cancelation.Done()
				return nil
			},
		}))
		loader := inputest.MustNewTestLoader(t, plugins, "type", "test")
		factory := RunnerFactory(
			log,
			beat.Info{Logger: log},
			monitoring.NewRegistry(),
			loader.Loader)

		runner, err := factory.Create(nil, conf.MustNewConfigFrom(map[string]interface{}{
			"type": "test",
		}))
		require.NoError(t, err)

		wg.Add(1)
		runner.Start()
		runner.Stop()
		wg.Wait()
		assert.Equal(t, 1, countRun)
	})

	t.Run("fail if input type is unknown to loader", func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		plugins := inputest.SinglePlugin("test", inputest.ConstInputManager(nil))
		loader := inputest.MustNewTestLoader(t, plugins, "type", "")
		factory := RunnerFactory(log, beat.Info{}, monitoring.NewRegistry(), loader.Loader)

		// run
		runner, err := factory.Create(nil, conf.MustNewConfigFrom(map[string]interface{}{
			"type": "unknown",
		}))
		assert.Nil(t, runner)
		assert.Error(t, err)
	})
}

func TestGenerateCheckConfig(t *testing.T) {
	tcs := []struct {
		name      string
		cfg       *conf.C
		want      *conf.C
		wantErr   error
		assertCfg func(t assert.TestingT, expected any, actual *conf.C, msgAndArgs ...any)
	}{
		{
			name: "id is present",
			cfg:  conf.MustNewConfigFrom("id: some-id"),
			assertCfg: func(t assert.TestingT, expect any, got *conf.C, msgAndArgs ...any) {
				id, _ := got.String("id", -1)
				if !strings.HasPrefix(id, "some-id") {
					t.Errorf("'id' field must start with the original id, got %q", id)
				}
				assert.NotEqual(t, expect, got, msgAndArgs)
			},
		},
		{
			name: "absent id",
			cfg:  conf.MustNewConfigFrom(""),
			assertCfg: func(t assert.TestingT, expect any, got *conf.C, msgAndArgs ...any) {
				if !got.HasField("id") {
					t.Errorf("expecting 'id' to be present in %s", conf.DebugString(got, true))
				}
				assert.NotNil(t, got, msgAndArgs...)
				assert.NotEqual(t, expect, got, msgAndArgs)
			},
		},
		{
			name:    "invalid config",
			cfg:     nil,
			wantErr: errors.New("failed to create new config"),
			assertCfg: func(t assert.TestingT, _ any, got *conf.C, msgAndArgs ...any) {
				assert.Nil(t, got, msgAndArgs...)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			f := factory{}

			got, err := f.generateCheckConfig(tc.cfg)
			if tc.wantErr != nil {
				assert.ErrorContains(t, err, tc.wantErr.Error())
			}

			tc.assertCfg(t, tc.cfg, got)
		})
	}
}
