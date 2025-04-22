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

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestNewPipelineClientListener_emptyReg(t *testing.T) {
	reg := monitoring.NewRegistry()

	listener := NewPipelineClientListener(reg, nil)
	require.NotNil(t, listener, "Listener should not be nil")

	pcl, ok := listener.(*PipelineClientListener)
	require.True(t, ok,
		"listener should be of type %T", &PipelineClientListener{})
	assert.NotNilf(t, pcl.eventsTotal,
		"%q metric should be created", metricEventsPipelineTotal)
	assert.NotNilf(t, pcl.eventsFiltered,
		"%q metric should be created", metricEventsPipelineFiltered)
	assert.NotNilf(t, pcl.eventsPublished,
		"%q metric should be created", metricEventsPipelinePublished)
}

func TestNewPipelineClientListener_reusedReg(t *testing.T) {
	reg := monitoring.NewRegistry()

	listener := NewPipelineClientListener(reg, nil)
	require.NotNil(t, listener, "Listener should not be nil")
	pcl, ok := listener.(*PipelineClientListener)
	require.True(t, ok,
		"listener should be of type %T", &PipelineClientListener{})
	gotTotal := pcl.eventsTotal
	gotFiltered := pcl.eventsFiltered
	gotPublished := pcl.eventsPublished

	assert.NotPanics(t, func() {
		// Call NewPipelineClientListener again reusing the metrics registry
		listener = NewPipelineClientListener(reg, nil)
	}, "Should not panic when reusing a metrics registry")
	require.NotNil(t, listener, "Listener should not be nil")

	pcl, ok = listener.(*PipelineClientListener)
	require.True(t, ok,
		"listener should be of type %T", &PipelineClientListener{})

	assert.Equalf(t, pcl.eventsTotal, gotTotal,
		"%q metric should have been reused", metricEventsPipelineTotal)
	assert.Equalf(t, pcl.eventsFiltered, gotFiltered,
		"%q metric should have been reused", metricEventsPipelineFiltered)
	assert.Equalf(t, pcl.eventsPublished, gotPublished,
		"%q metric should have been reused", metricEventsPipelinePublished)
}

func TestNewPipelineClientListener_ClientListener(t *testing.T) {
	tcs := []struct {
		name   string
		cl     beat.ClientListener
		assert func(*testing.T, any)
	}{
		{
			name: "nil clientListener",
			cl:   nil,
			assert: func(t *testing.T, got any) {
				assert.IsTypef(t, &PipelineClientListener{}, got,
					"want %T, got %T", &PipelineClientListener{}, got)
			},
		},
		{
			name: "existing clientListener",
			cl:   &PipelineClientListener{},
			assert: func(t *testing.T, got any) {
				assert.IsTypef(t, &beat.CombinedClientListener{}, got,
					"want %T, got %T", &PipelineClientListener{}, got)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reg := monitoring.NewRegistry()
			got := NewPipelineClientListener(reg, tc.cl)
			assert.NotNil(t, got, "ClientListener should not be nil")
			tc.assert(t, got)
		})
	}
}
