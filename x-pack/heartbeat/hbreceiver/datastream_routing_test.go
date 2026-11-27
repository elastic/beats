// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// The receiver test binary does not link the real browser monitor, which needs
// node anyway. These stand-ins emit what synthexec emits for one journey: a
// root event plus continuations tagged with a sub-dataset (see
// x-pack/heartbeat/monitors/browser/synthexec/enrich.go).
var fakeSyntheticsPlugins = map[string][]string{
	"browser": {"browser.network", "browser.screenshot"},
	"api":     {"api.network"},
}

func init() {
	for monitorType, datasets := range fakeSyntheticsPlugins {
		if _, exists := plugin.GlobalPluginsReg.Get(monitorType); exists {
			panic(monitorType + " monitor is already registered; the fake routing plugin would shadow nothing")
		}
		plugin.Register(monitorType, fakeSyntheticsMake(datasets))
	}
}

func fakeSyntheticsMake(datasets []string) plugin.PluginMake {
	return func(_ string, _ *config.C, _ beat.Info) (plugin.Plugin, error) {
		// Like synthexec, each event's job returns the next one, so the summarizer
		// sees them sequentially rather than as parallel continuations.
		var next func(i int) jobs.Job
		next = func(i int) jobs.Job {
			return func(event *beat.Event) ([]jobs.Job, error) {
				if i > 0 {
					ds := datasets[i-1]
					add_data_stream.SetEventDataset(event, ds)
					_, _ = event.PutValue("fake.sub_dataset", ds)
				}
				if i == len(datasets) {
					return nil, nil
				}
				return []jobs.Job{next(i + 1)}, nil
			}
		}
		return plugin.Plugin{Jobs: []jobs.Job{next(0)}, Endpoints: 1}, nil
	}
}

type routedEvent struct {
	index string
	body  mapstr.M
}

// TestSyntheticsSubDatasetRouting checks that browser/api sub-dataset events
// (network, screenshot) keep their own data stream in receiver mode, as they do
// in process mode.
func TestSyntheticsSubDatasetRouting(t *testing.T) {
	for monitorType, subDatasets := range fakeSyntheticsPlugins {
		monitorSource := map[string]any{
			"id":       "my-" + monitorType + "-monitor",
			"name":     "My " + monitorType + " monitor",
			"type":     monitorType,
			"schedule": "@every 1h",
			"enabled":  true,
			"origin":   "ui",
		}

		configs := map[string]map[string]any{
			// What the process runtime ends up with after agentconfig.Transform:
			// the stream source plus its data_stream, no index and no agent
			// metadata processor.
			"process-equivalent": withDataStream(monitorSource, monitorType),
			// What elastic-agent's OTel translation hands the receiver
			// (internal/pkg/otel/translate/otelconfig.go getInputsForUnit).
			"otel-translated": agentTranslatedMonitor(t, monitorSource, monitorType),
		}

		for name, monitor := range configs {
			t.Run(monitorType+"/"+name, func(t *testing.T) {
				events := runMonitorOnce(t, monitor)

				for _, ds := range subDatasets {
					ev, ok := findSubDatasetEvent(events(), ds)
					require.Truef(t, ok, "no %s event received", ds)

					want := "synthetics-" + ds + "-default"
					assert.Equalf(t, want, ev.index, "%s event must be written to its own data stream", ds)
					flat := ev.body.Flatten()
					assert.Equalf(t, ds, flat["data_stream.dataset"], "%s event data_stream.dataset", ds)
					assert.Equalf(t, ds, flat["event.dataset"], "%s event event.dataset", ds)
				}
			})
		}
	}
}

func withDataStream(source map[string]any, monitorType string) map[string]any {
	m := mapstr.M(source).Clone()
	m["data_stream"] = map[string]any{"type": "synthetics", "dataset": monitorType, "namespace": "default"}
	return m
}

func agentTranslatedMonitor(t *testing.T, source map[string]any, monitorType string) map[string]any {
	t.Helper()
	src, err := structpb.NewStruct(source)
	require.NoError(t, err, "building stream source")

	unit := &proto.UnitExpectedConfig{
		Id:         "synthetics/" + monitorType + "-my-monitor",
		Type:       "synthetics/" + monitorType,
		Name:       "synthetics-1",
		DataStream: &proto.DataStream{Namespace: "default"},
		Streams: []*proto.Stream{{
			Id:         monitorType + "-my-monitor",
			DataStream: &proto.DataStream{Type: "synthetics", Dataset: monitorType},
			Source:     src,
		}},
	}
	inputs, err := management.CreateInputsFromStreamsForReceiver(unit, "logs",
		&client.AgentInfo{ID: "agent-id", Version: "9.9.0"})
	require.NoError(t, err, "translating unit like elastic-agent does")
	require.Len(t, inputs, 1, "one monitor per stream")
	return inputs[0]
}

func runMonitorOnce(t *testing.T, monitor map[string]any) func() []routedEvent {
	t.Helper()

	var mu sync.Mutex
	var got []routedEvent
	sink, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
		mu.Lock()
		defer mu.Unlock()
		for _, rl := range ld.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					idx, _ := lr.Attributes().Get("elasticsearch.index")
					got = append(got, routedEvent{index: idx.Str(), body: lr.Body().Map().AsRaw()})
				}
			}
		}
		return nil
	})
	require.NoError(t, err, "creating log sink")

	cfg := &Config{Beatconfig: map[string]any{
		"heartbeat":               map[string]any{"monitors": []any{monitor}},
		"queue.mem.flush.timeout": "0s",
		"path.home":               t.TempDir(),
	}}
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	set := receiver.Settings{
		ID: component.NewIDWithName(factory.Type(), t.Name()),
		// Not zaptest: monitors/logger keeps a process-wide logger, which would
		// outlive this test and panic on the next subtest's run log.
		Logger: zap.NewNop(),
	}
	monitorType, ok := monitor["type"].(string)
	require.True(t, ok, "monitor type must be a string")

	rec, err := factory.CreateLogs(t.Context(), set, cfg, sink)
	require.NoError(t, err, "creating receiver")
	require.NoError(t, rec.Start(t.Context(), componenttest.NewNopHost()), "starting receiver")
	t.Cleanup(func() { assert.NoError(t, rec.Shutdown(context.Background()), "stopping receiver") })

	snapshot := func() []routedEvent {
		mu.Lock()
		defer mu.Unlock()
		return append([]routedEvent(nil), got...)
	}
	require.Eventually(t, func() bool {
		for _, ds := range fakeSyntheticsPlugins[monitorType] {
			if _, ok := findSubDatasetEvent(snapshot(), ds); !ok {
				return false
			}
		}
		return true
	}, 30*time.Second, 100*time.Millisecond, "monitor did not emit its sub-dataset events")
	return snapshot
}

func findSubDatasetEvent(events []routedEvent, ds string) (routedEvent, bool) {
	for _, ev := range events {
		if v, _ := ev.body.GetValue("fake.sub_dataset"); v == ds {
			return ev, true
		}
	}
	return routedEvent{}, false
}
