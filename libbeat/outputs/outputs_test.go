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

package outputs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/mock-es/pkg/api"
)

func TestOutputsMetrics(t *testing.T) {
	t.Run("elasticsearch", func(t *testing.T) {
		rdr := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(rdr))
		mockESHandler := api.NewDeterministicAPIHandler(
			uuid.Must(uuid.NewV4()),
			"",
			provider,
			time.Now().Add(24*time.Hour),
			0,
			100,
			func(action api.Action, event []byte) int {
				ev := map[string]string{}
				err := json.Unmarshal(event, &ev)
				if err != nil {
					t.Errorf("failed to unmarshal event: %v", err)
					return http.StatusInternalServerError
				}

				t.Logf("\naction: %v\n\tmeta: %s\n\tevent: %s", action.Action, action.Meta, string(event))

				httpStatus, err := strconv.Atoi(ev["http_status"])
				if err != nil {
					t.Errorf("failed to parse %s to int: %v", ev["http_status"], err)
					return http.StatusInternalServerError
				}
				return httpStatus
			})

		esMock := httptest.NewServer(mockESHandler)
		reg := monitoring.NewRegistry()
		rawCfg := fmt.Sprintf(`{"hosts": ["%s"]}`, esMock.URL)
		cfg, err := config.NewConfigFrom(rawCfg)
		require.NoError(t, err, "could not parse config")

		factory := outputs.FindFactory("elasticsearch")
		im := mockIndexManager("mock-index")
		beatInfo := beat.Info{Logger: logptest.NewTestingLogger(t,
			"elasticsearch",
			zap.WithCaller(false))}

		og, err := factory(im, beatInfo, outputs.NewStats(reg), cfg)
		require.NoError(t, err, "could not create output group")

		counter := &beat.CountOutputListener{}
		observer := publisher.OutputListener{Listener: counter}
		evs := []publisher.Event{
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":         "message 1",
						"http_status": strconv.Itoa(http.StatusOK)},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg": "message 2",
						// dropped
						"http_status": strconv.Itoa(http.StatusNotAcceptable)},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg": "message 3",
						// toomany
						"http_status": strconv.Itoa(http.StatusTooManyRequests)},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg": "message 4",
						// duplicated
						"http_status": strconv.Itoa(http.StatusConflict)},
					Private:    nil,
					TimeSeries: false,
				},
			},
		}

		// Try publishing a batch that can be split
		batch := mockBatch(evs)
		err = og.Clients[0].Publish(context.Background(), batch)
		require.NoError(t, err, "could not publish events")

		snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)
		events := snapshot["events"].(map[string]any)

		evAcked := events["acked"].(int64)
		evNew := events["total"].(int64)
		evDropped := events["dropped"].(int64)
		evDeadLetter := events["dead_letter"].(int64)
		evDuplicated := events["duplicates"].(int64)
		evTooMany := events["toomany"].(int64)
		evRetrieable := events["failed"].(int64)

		assert.Equal(t, evNew, counter.NewLoad())
		assert.Equal(t, evAcked, counter.AckedLoad())
		assert.Equal(t, evDropped, counter.DroppedLoad())
		assert.Equal(t, evDeadLetter, counter.DeadLetterLoad())
		assert.Equal(t, evDuplicated, counter.DuplicateEventsLoad())
		assert.Equal(t, evTooMany, counter.ErrTooManyLoad())
		assert.Equal(t, evRetrieable, counter.RetryableErrorsLoad())

		if t.Failed() {
			t.Log("OutputListener metrics: ", counter)

			snapshotJson, err := json.Marshal(snapshot)
			require.NoErrorf(t, err,
				"could not marshal metrics snapshot. Raw metrics snapshot: %v",
				snapshot)
			t.Logf("metrics registry snapshot: %s", snapshotJson)
		}

	})
}

var _ outputs.IndexManager = (*mockIndexManager)(nil)

type mockIndexManager string

func (m mockIndexManager) BuildSelector(*config.C) (outputs.IndexSelector, error) {
	return mockIndexSelector(m), nil
}

var _ outputs.IndexSelector = (*mockIndexSelector)(nil)

type mockIndexSelector string

func (m mockIndexSelector) Select(*beat.Event) (string, error) {
	return string(m), nil
}

type mockBatch []publisher.Event

func (m mockBatch) Events() []publisher.Event {
	return m
}

func (m mockBatch) ACK() {
}

func (m mockBatch) Drop() {
}

func (m mockBatch) Retry() {
}

func (m mockBatch) RetryEvents(events []publisher.Event) {
	m = events
}

func (m mockBatch) SplitRetry() bool { return false }

func (m mockBatch) Cancelled() {
}
