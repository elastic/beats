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

package beater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type fakeElasticsearchRequester struct {
	requestFn func(method, path, pipeline string, params map[string]string, body any) (int, []byte, error)
}

func (f fakeElasticsearchRequester) Request(method, path, pipeline string, params map[string]string, body any) (int, []byte, error) {
	return f.requestFn(method, path, pipeline, params, body)
}

func TestHeartbeatWithElasticsearchStateLoader(t *testing.T) {
	logger := logp.NewNopLogger()
	stateLoader, replaceStateLoader := monitorstate.AtomicStateLoader(monitorstate.NilStateLoader, logger)

	bt := &Heartbeat{
		config:             &config.Config{},
		replaceStateLoader: replaceStateLoader,
		logger:             logger,
	}

	var requestCount int
	fake := fakeElasticsearchRequester{
		requestFn: func(string, string, string, map[string]string, any) (int, []byte, error) {
			requestCount++
			return 200, []byte(`{"hits":{"hits":[]}}`), nil
		},
	}

	bt.WithElasticsearchStateLoader(fake)

	_, err := stateLoader(stdfields.StdMonitorFields{ID: "mon-1", Type: "http"})
	require.NoError(t, err, "installed ES loader should succeed with empty hits")
	assert.Equal(t, 1, requestCount, "installed loader should call the injected requester")
}

func TestMakeESClient(t *testing.T) {
	t.Run("should not modify the timeout setting from original config", func(t *testing.T) {
		origTimeout := 90
		origCfg, _ := conf.NewConfigFrom(map[any]any{
			"hosts":    []string{"http://localhost:9200"},
			"username": "anyuser",
			"password": "anypwd",
			"timeout":  origTimeout,
		})
		anyAttempt := 1
		anyDuration := 1 * time.Second

		_, _ = makeESClient(context.Background(), origCfg, anyAttempt, anyDuration, logptest.NewTestingLogger(t, ""), beat.Info{})

		timeout, err := origCfg.Int("timeout", -1)
		require.NoError(t, err)
		assert.EqualValues(t, origTimeout, timeout)
	})
}
