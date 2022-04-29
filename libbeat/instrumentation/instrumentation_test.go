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

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestInstrumentationConfig(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
		},
	})
	instrumentation, err := New(cfg, "my-beat", version.GetDefaultVersion())
	require.NoError(t, err)

	tracer := instrumentation.Tracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.NotNil(t, instrumentation.Listener())
}

func TestInstrumentationConfigExplicitHosts(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
			"hosts":   []string{"localhost:8200"},
		},
	},
	)
	instrumentation, err := New(cfg, "my-beat", version.GetDefaultVersion())
	require.NoError(t, err)
	tracer := instrumentation.Tracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.Nil(t, instrumentation.Listener())
}

func TestInstrumentationConfigListener(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
		},
	})
	instrumentation, err := New(cfg, "apm-server", version.GetDefaultVersion())
	require.NoError(t, err)

	tracer := instrumentation.Tracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.NotNil(t, instrumentation.Listener())
}

func TestAPMTracerDisabledByDefault(t *testing.T) {
	instrumentation, err := New(config.NewConfig(), "beat", "8.0")
	require.NoError(t, err)
	tracer := instrumentation.Tracer()
	require.NotNil(t, tracer)
	assert.False(t, tracer.Active())
}

func TestInstrumentationDisabled(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "false",
		},
	})
	instrumentation, err := New(cfg, "filebeat", version.GetDefaultVersion())
	require.NoError(t, err)
	require.NotNil(t, instrumentation)

}
