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

package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestOTelQueueMetrics(t *testing.T) {
	// More thorough testing of queue metrics are in the queue package,
	// here we just want to make sure that they appear under the right
	// monitoring namespace.
	reg := monitoring.NewRegistry()
	logger := logptest.NewTestingLogger(t, "")
	controller, err := newOTelOutputController(
		beat.Info{Logger: logger},
		Monitors{
			Logger:  logger,
			Metrics: reg,
		},
		nilObserver,
		memqueue.FactoryForSettings[publisher.Event](memqueue.Settings{Events: 1000}),
		"")
	require.NoError(t, err, "creating OTel output controller should succeed")
	defer controller.waitClose(context.Background(), true)
	entry := reg.Get("pipeline.queue.max_events")
	require.NotNil(t, entry, "pipeline.queue.max_events must exist")
	value, ok := entry.(*monitoring.Uint)
	require.True(t, ok, "pipeline.queue.max_events must be a *monitoring.Uint")
	assert.Equal(t, uint64(1000), value.Get(), "pipeline.queue.max_events should match the events configuration key")
}
