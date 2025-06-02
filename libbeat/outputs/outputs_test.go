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
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/v7/libbeat/outputs/kafka"
	_ "github.com/elastic/beats/v7/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/v7/libbeat/outputs/redis"
	"github.com/elastic/beats/v7/libbeat/publisher"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/outputs/otelconsumer"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/mock-es/pkg/api"
)

func TestOutputsMetrics(t *testing.T) {
	defaultEvFields := []map[string]any{
		{"msg": "message 1"},
		{"msg": "message 2"},
		{"msg": "message 3"},
		{"msg": "message 4"},
	}

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

				httpStatus, err := strconv.Atoi(ev["http_status"])
				if err != nil {
					t.Errorf("failed to parse %s to int: %v", ev["http_status"], err)
					return http.StatusInternalServerError
				}
				return httpStatus
			})

		esMock := httptest.NewServer(mockESHandler)
		rawCfg := map[string]any{"hosts": []string{esMock.URL}}

		evFields := []map[string]any{
			{
				"msg":         "message 1",
				"http_status": strconv.Itoa(http.StatusOK),
			},
			{
				"msg": "message 2",
				// dropped
				"http_status": strconv.Itoa(http.StatusNotAcceptable),
			},
			{
				"msg": "message 3",
				// toomany
				"http_status": strconv.Itoa(http.StatusTooManyRequests),
			},
			{
				"msg": "message 4",
				// duplicated
				"http_status": strconv.Itoa(http.StatusConflict),
			},
		}

		testOutputMetrics(t, "elasticsearch", rawCfg, evFields)
	})

	t.Run("otelconsumer", func(t *testing.T) {
		testOutputMetrics(t, "otelconsumer", nil, defaultEvFields)
	})
}

func testOutputMetrics(t *testing.T,
	output string,
	configuration any,
	evFields []map[string]any) {

	counter := &beat.CountOutputListener{}
	observer := publisher.OutputListener{Listener: counter}

	log, logBuff := logp.NewInMemoryLocal(output, logp.ConsoleEncoderConfig())
	defer func() {
		if t.Failed() {
			t.Logf("\n%s", logBuff.String())
		}
	}()

	lconsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		return nil
	})
	require.NoError(t, err)
	beatInfo := beat.Info{
		Logger:      log,
		LogConsumer: lconsumer,
	}
	reg := monitoring.NewRegistry()

	cfg, err := config.NewConfigFrom(configuration)
	require.NoError(t, err, "could not parse config")
	factory := outputs.FindFactory(output)
	og, err := factory(
		mockIndexManager("mock-index"),
		beatInfo,
		outputs.NewStats(reg),
		cfg)
	require.NoError(t, err, "could not create output group")

	var evs []publisher.Event
	for _, fields := range evFields {
		ev := publisher.Event{
			Content: beat.Event{
				Timestamp:  time.Time{},
				Meta:       nil,
				Fields:     fields,
				Private:    nil,
				TimeSeries: false,
			},
			OutputListener: observer,
		}

		if og.EncoderFactory != nil {
			encoderFactory := og.EncoderFactory()
			e, _ := encoderFactory.EncodeEntry(ev)
			ev = e.(publisher.Event)
		}
		evs = append(evs, ev)
	}

	client := og.Clients[0]
	if connectable, ok := client.(outputs.Connectable); ok {
		require.NoError(t, connectable.Connect(context.Background()),
			"could not connect %s", client.String())
	}

	// Try publishing a batch that can be split
	err = client.Publish(context.Background(), &mockBatch{evs: evs})
	require.NoError(t, err, "could not publish events")
	require.NoError(t, og.Clients[0].Close(), "failed to close output client")

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

	t.Logf("output metrics: %s", counter)
	if t.Failed() {
		t.Log("OutputListener metrics: ", counter)

		snapshotJson, err := json.Marshal(snapshot)
		require.NoErrorf(t, err,
			"could not marshal metrics snapshot. Raw metrics snapshot: \n%v",
			snapshot)
		t.Logf("metrics registry snapshot: \n%s", snapshotJson)
	}
}

func getenv(name, defaultValue string) string {
	v := os.Getenv(name)
	if v == "" {
		return defaultValue
	}
	return v
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

var _ publisher.Batch = (*mockBatch)(nil)

type mockBatch struct {
	evs []publisher.Event
}

func (m *mockBatch) Events() []publisher.Event {
	return m.evs
}

func (m *mockBatch) ACK() {
}

func (m *mockBatch) Drop() {
}

func (m *mockBatch) Retry() {
}

func (m *mockBatch) RetryEvents(events []publisher.Event) {
	m.evs = events
}

func (m *mockBatch) SplitRetry() bool { return false }

func (m *mockBatch) Cancelled() {}
