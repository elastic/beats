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

package beat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/version"
)

func TestAPMTracerDisabledByDefault(t *testing.T) {
	b := Beat{
		Info: Info{
			Beat:        "my-beat",
			IndexPrefix: "my-beat-*",
			Version:     version.GetDefaultVersion(),
			Name:        "my-beat",
		},
	}
	tracer := b.Instrumentation.GetTracer()
	require.NotNil(t, tracer)
	defer tracer.Close()
	assert.False(t, tracer.Active())
}

func TestInstrumentationConfig(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
		},
	})
	instrumentation, err := CreateInstrumentation(cfg, Info{Name: "my-beat", Version: version.GetDefaultVersion()})
	require.NoError(t, err)

	tracer := instrumentation.GetTracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.Nil(t, instrumentation.Listener)
}

func TestInstrumentationConfigExplicitHosts(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
			"hosts":   []string{"localhost:8200"},
		},
	},
	)
	instrumentation, err := CreateInstrumentation(cfg, Info{Name: "my-beat", Version: version.GetDefaultVersion()})
	require.NoError(t, err)
	tracer := instrumentation.GetTracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.Nil(t, instrumentation.Listener)
}

func TestInstrumentationConfigListener(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"instrumentation": map[string]interface{}{
			"enabled": "true",
		},
	})
	instrumentation, err := CreateInstrumentation(cfg, Info{Name: "apm-server", Version: version.GetDefaultVersion()})
	require.NoError(t, err)

	tracer := instrumentation.GetTracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())
	assert.NotNil(t, instrumentation.Listener)
}
