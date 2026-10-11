// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions" // Registers the add_fields processor used by inputWithNamespace.
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type recordingClient struct {
	events []beat.Event
}

func (c *recordingClient) Publish(event beat.Event) {
	c.events = append(c.events, event)
}

func (c *recordingClient) PublishAll(events []beat.Event) {
	c.events = append(c.events, events...)
}

func (c *recordingClient) Close() error {
	return nil
}

func TestPublishScheduledResponseTimestamp(t *testing.T) {
	completedAt := time.Date(2024, 1, 1, 4, 30, 0, 0, time.UTC)
	tests := []struct {
		name                string
		plannedScheduleTime time.Time
		expectedTimestamp   time.Time
	}{
		{
			name:                "uses completion time after planned slot",
			plannedScheduleTime: completedAt.Add(-time.Minute),
			expectedTimestamp:   completedAt,
		},
		{
			name:                "floors timestamp at future planned slot",
			plannedScheduleTime: completedAt.Add(30 * time.Minute),
			expectedTimestamp:   completedAt.Add(30 * time.Minute),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &recordingClient{}
			publisher := &Publisher{
				log:                   logptest.NewTestingLogger(t, "scheduled_response"),
				actionResponsesClient: client,
			}

			publisher.PublishScheduledResponse(
				"schedule-id",
				"",
				"",
				"",
				"",
				"response-id",
				completedAt.Add(-time.Second),
				completedAt,
				tc.plannedScheduleTime,
				3,
				1,
			)

			require.Len(t, client.events, 1, "scheduled response should publish one event")
			event := client.events[0]
			assert.Equal(t, tc.expectedTimestamp, event.Timestamp, "scheduled response event timestamp should not precede its planned slot")
			assert.Equal(t, completedAt.Format(time.RFC3339Nano), event.Fields["completed_at"], "completed_at should preserve the endpoint time")
			assert.Equal(t, tc.plannedScheduleTime.Format(time.RFC3339Nano), event.Fields["planned_schedule_time"], "planned_schedule_time should preserve the computed slot")
		})
	}
}

func TestHitToEvent(t *testing.T) {

	const maxMask = 0b1111111

	type params struct {
		index, eventType, idValue, idFieldKey, responseID string
		meta                                              map[string]any
		hit                                               map[string]any
		ecsm                                              ecs.Mapping
		reqData                                           any
	}

	genParams := func(mask int) (p params) {
		if mask>>6&1 > 0 {
			p.index = "logs-osquery_manager.result-default"
		}
		if mask>>5&1 > 0 {
			p.eventType = "osquery_manager"
		}
		if mask>>4&1 > 0 {
			p.idValue = "uptime"
			p.idFieldKey = "action_id"
		}
		if mask>>3&1 > 0 {
			p.responseID = uuid.Must(uuid.NewV4()).String()
		}
		if mask>>2&1 > 0 {
			p.hit = map[string]any{
				"foo": "bar",
			}
		}
		if mask>>1&1 > 0 {
			p.ecsm = ecs.Mapping{
				"foo": ecs.MappingInfo{
					Field: "food",
				},
			}
		}
		if mask&1 > 0 {
			p.reqData = map[string]any{
				"query": "select * from uptime",
			}
		}
		return p
	}

	for i := range maxMask {
		p := genParams(i)
		ev := hitToEvent(p.index, p.eventType, p.idValue, p.idFieldKey, p.responseID, "", "", "", "", p.meta, p.hit, p.ecsm, p.reqData)

		if p.index != "" {
			diff := cmp.Diff(p.index, ev.Meta[events.FieldMetaRawIndex])
			if diff != "" {
				t.Error(diff)
			}
		} else {
			if ev.Meta != nil {
				t.Error("expected ev.Meta nil")
			}
		}

		diff := cmp.Diff(p.eventType, ev.Fields["type"])
		if diff != "" {
			t.Error(diff)
		}

		if p.idFieldKey != "" {
			diff = cmp.Diff(p.idValue, ev.Fields[p.idFieldKey])
			if diff != "" {
				t.Error(diff)
			}
		} else if ev.Fields["action_id"] != nil || ev.Fields["schedule_id"] != nil {
			t.Error("expected no id field when key is empty")
		}

		if p.responseID != "" {
			diff := cmp.Diff(p.responseID, ev.Fields["response_id"])
			if diff != "" {
				t.Error(diff)
			}
		} else {
			if ev.Fields["response_id"] != nil {
				t.Error(`expected ev.Fields["response_id"] nil`)
			}
		}

		diff = cmp.Diff(p.hit, ev.Fields["osquery"])
		if diff != "" {
			t.Error(diff)
		}

		diff = cmp.Diff(p.reqData, ev.Fields["action_data"])
		if diff != "" {
			t.Error(diff)
		}

		// Should be close to the current time, set to time.Hour in case of debugging for example
		if time.Since(ev.Timestamp) > time.Hour {
			t.Errorf("unexpected ev.Timestamp: %v", ev.Timestamp)
		}
	}
}

func TestActionResultToEvent(t *testing.T) {

	tests := []struct {
		name     string
		req, res map[string]any
		want     map[string]any
	}{
		{
			name: "successful",
			req: toMap(t, `{
				"data": {
					"id": "a72d65d8-200a-4b43-8dbd-7bc0e9ce8e65",
					"query": "select * from osquery_info"
				},
				"id": "5c433f88-ab0d-41e2-af76-6ff16ae3ced8",
				"input_type": "osquery",
				"type": "INPUT_ACTION"
			}`),
			res: toMap(t, `{
				"completed_at": "2024-04-18T19:39:39.740162Z",
				"count": 1,
				"started_at": "2024-04-18T19:39:39.532125Z"
			} `),
			want: toMap(t, `{
				"completed_at": "2024-04-18T19:39:39.740162Z",
				"action_response": {
					"osquery": {
						"count": 1
					}
				},
				"action_id": "5c433f88-ab0d-41e2-af76-6ff16ae3ced8",
				"started_at": "2024-04-18T19:39:39.532125Z",
				"action_input_type": "osquery",
				"action_data": {
					"id": "a72d65d8-200a-4b43-8dbd-7bc0e9ce8e65",
					"query": "select * from osquery_info"
				}
			}`),
		},
		{
			name: "error",
			req: toMap(t, `{
				"data": {
					"id": "08995ee8-5182-423e-9527-552736411010",
					"query": "select * from osquery_foo"
				},
				"id": "70539d80-4082-41e9-aff4-fbb877dd752b",
				"input_type": "osquery",
				"type": "INPUT_ACTION"
			}`),
			res: toMap(t, `{
				"completed_at": "2024-04-20T14:56:34.87195Z",
				"error": "query failed, code: 1, message: no such table: osquery_foo",
				"started_at": "2024-04-20T14:56:34.87195Z"
			}`),
			want: toMap(t, `{
				"completed_at": "2024-04-20T14:56:34.87195Z",
				"action_id": "70539d80-4082-41e9-aff4-fbb877dd752b",
				"started_at": "2024-04-20T14:56:34.87195Z",
				"action_input_type": "osquery",
				"error": "query failed, code: 1, message: no such table: osquery_foo",
				"action_data": {
				  "id": "08995ee8-5182-423e-9527-552736411010",
				  "query": "select * from osquery_foo"
				}
			  }`),
		},
		{
			name: "successful with space id",
			req: toMap(t, `{
				"data": {
					"id": "a72d65d8-200a-4b43-8dbd-7bc0e9ce8e65",
					"query": "select * from osquery_info"
				},
				"id": "5c433f88-ab0d-41e2-af76-6ff16ae3ced8",
				"input_type": "osquery",
				"space_id": "production",
				"type": "INPUT_ACTION"
			}`),
			res: toMap(t, `{
				"completed_at": "2024-04-18T19:39:39.740162Z",
				"count": 1,
				"started_at": "2024-04-18T19:39:39.532125Z"
			} `),
			want: toMap(t, `{
				"completed_at": "2024-04-18T19:39:39.740162Z",
				"action_response": {
					"osquery": {
						"count": 1
					}
				},
				"action_id": "5c433f88-ab0d-41e2-af76-6ff16ae3ced8",
				"started_at": "2024-04-18T19:39:39.532125Z",
				"action_input_type": "osquery",
				"action_data": {
					"id": "a72d65d8-200a-4b43-8dbd-7bc0e9ce8e65",
					"query": "select * from osquery_info"
				},
				"space_id": "production"
			}`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := actionResultToEvent(tc.req, tc.res)
			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestHitToEvent_SpaceID(t *testing.T) {
	spaceID := "space-abc"
	ev := hitToEvent(
		"logs-osquery_manager.result-default",
		"osquery_manager",
		"sched-123",
		"schedule_id",
		uuid.Must(uuid.NewV4()).String(),
		spaceID,
		"",
		"",
		"",
		nil,
		map[string]any{"foo": "bar"},
		nil,
		nil,
	)

	if diff := cmp.Diff(spaceID, ev.Fields["space_id"]); diff != "" {
		t.Error(diff)
	}
}

func TestHitToEvent_PackID(t *testing.T) {
	packID := "pack-xyz"
	ev := hitToEvent(
		"logs-osquery_manager.result-default",
		"osquery_manager",
		"sched-123",
		"schedule_id",
		uuid.Must(uuid.NewV4()).String(),
		"",
		packID,
		"",
		"",
		nil,
		map[string]any{"foo": "bar"},
		nil,
		nil,
	)

	if diff := cmp.Diff(packID, ev.Fields["pack_id"]); diff != "" {
		t.Error(diff)
	}
}

func TestHitToEvent_PackNameAndQueryName(t *testing.T) {
	packName := "My Pack"
	queryName := "processes"
	ev := hitToEvent(
		"logs-osquery_manager.result-default",
		"osquery_manager",
		"sched-123",
		"schedule_id",
		uuid.Must(uuid.NewV4()).String(),
		"",
		"pack-xyz",
		packName,
		queryName,
		nil,
		map[string]any{"foo": "bar"},
		nil,
		nil,
	)

	if diff := cmp.Diff(packName, ev.Fields["pack_name"]); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(queryName, ev.Fields["query_name"]); diff != "" {
		t.Error(diff)
	}
}

func TestHitToEvent_NoPackNameOrQueryName(t *testing.T) {
	ev := hitToEvent(
		"logs-osquery_manager.result-default",
		"osquery_manager",
		"action-123",
		"action_id",
		uuid.Must(uuid.NewV4()).String(),
		"",
		"",
		"",
		"",
		nil,
		map[string]any{"foo": "bar"},
		nil,
		nil,
	)

	if _, ok := ev.Fields["pack_name"]; ok {
		t.Error(`expected no "pack_name" field when packName is empty`)
	}
	if _, ok := ev.Fields["query_name"]; ok {
		t.Error(`expected no "query_name" field when queryName is empty`)
	}
}

func TestQueryProfileToEvent_SpaceID(t *testing.T) {
	tests := []struct {
		name    string
		spaceID string
		present bool
	}{
		{name: "present", spaceID: "production", present: true},
		{name: "absent", spaceID: "", present: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fields := queryProfileToEvent("", "", "", tc.spaceID, map[string]any{"key": "val"}, nil)
			got, ok := fields["space_id"]
			if tc.present {
				if !ok {
					t.Errorf("expected space_id %q, field not present", tc.spaceID)
				} else if got != tc.spaceID {
					t.Errorf("space_id mismatch: got=%q want=%q", got, tc.spaceID)
				}
			} else {
				if ok {
					t.Errorf("expected no space_id field, got %v", got)
				}
			}
		})
	}
}

// TestConfigureByDataset verifies that Configure builds each client from the
// input for its dataset rather than from the input at its position, so
// out-of-order delivery does not break client binding.
func TestConfigureByDataset(t *testing.T) {
	result := inputWithNamespace(config.DefaultDataset, "ns-result")
	actions := inputWithNamespace(config.DefaultActionResponsesDataset, "ns-actions")
	profile := inputWithNamespace(config.DefaultQueryProfileDataset, "ns-profile")

	tests := []struct {
		name   string
		inputs []config.InputConfig
	}{
		{name: "result_first", inputs: []config.InputConfig{result, actions, profile}},
		{name: "reversed", inputs: []config.InputConfig{profile, actions, result}},
		{name: "result_last", inputs: []config.InputConfig{actions, profile, result}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := &connectRecorder{}
			p := New(&beat.Beat{Publisher: rec}, logptest.NewTestingLogger(t, "pub"))
			if err := p.Configure(test.inputs); err != nil {
				t.Fatalf("Configure: %v", err)
			}

			for _, tc := range []struct {
				name   string
				client beat.Client
				want   string
			}{
				{"client", p.client, "ns-result"},
				{"actionResponsesClient", p.actionResponsesClient, "ns-actions"},
				{"queryProfileClient", p.queryProfileClient, "ns-profile"},
			} {
				if tc.client == nil {
					t.Errorf("%s is nil", tc.name)
					continue
				}
				if got := rec.namespaceOf(t, tc.client); got != tc.want {
					t.Errorf("%s data_stream.namespace = %q; want %q", tc.name, got, tc.want)
				}
			}
		})
	}
}

// inputWithNamespace returns an input for the dataset that has a processor, so
// that Configure adds a data_stream processor carrying the namespace. The
// namespace identifies which input a client was built from.
func inputWithNamespace(dataset, namespace string) config.InputConfig {
	return config.InputConfig{
		Datastream: config.DatastreamConfig{Dataset: dataset, Type: "logs", Namespace: namespace},
		Processors: processors.PluginConfig{
			conf.MustNewConfigFrom(map[string]any{
				"add_fields": map[string]any{"target": "", "fields": map[string]any{"a": "b"}},
			}),
		},
	}
}

// connectRecorder is a beat.Pipeline that records the configuration of each
// client it creates.
type connectRecorder struct {
	configs []beat.ClientConfig
	clients []*recordingClient
}

func (r *connectRecorder) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	c := &recordingClient{}
	r.configs = append(r.configs, cfg)
	r.clients = append(r.clients, c)
	return c, nil
}

func (r *connectRecorder) Connect() (beat.Client, error) {
	return r.ConnectWith(beat.ClientConfig{})
}

func (r *connectRecorder) Disconnect(context.Context) error { return nil }

// namespaceOf returns the data_stream.namespace that the processors of the
// client add to an event, or the empty string if the client was not created by r
// or its processors set no data_stream.
func (r *connectRecorder) namespaceOf(t *testing.T, client beat.Client) string {
	t.Helper()
	for i, c := range r.clients {
		if beat.Client(c) != client {
			continue
		}
		ev, err := r.configs[i].Processing.Processor.Run(&beat.Event{})
		if err != nil {
			t.Fatalf("running processors: %v", err)
		}
		ds, ok := ev.Fields["data_stream"].(add_data_stream.DataStream)
		if !ok {
			return ""
		}
		return ds.Namespace
	}
	return ""
}

// TestConfigureCustomDataset verifies that an input with a dataset that is not
// one of the known streams is still bound as the result stream, so that
// Publish has a client.
func TestConfigureCustomDataset(t *testing.T) {
	p := New(&beat.Beat{Publisher: pipeline.NewNilPipeline()}, logptest.NewTestingLogger(t, "pub"))

	inputs := []config.InputConfig{
		{Datastream: config.DatastreamConfig{Dataset: "custom.dataset", Type: "logs", Namespace: "default"}},
	}
	if err := p.Configure(inputs); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if p.client == nil {
		t.Error("p.client is nil; Publish would panic")
	}
}

// TestConfigureEmptyDataset verifies that an input with no explicit dataset is
// treated as the result stream, as processorsForInputConfig does when it
// defaults an empty dataset.
func TestConfigureEmptyDataset(t *testing.T) {
	b := &beat.Beat{
		Publisher: pipeline.NewNilPipeline(),
	}
	p := New(b, logptest.NewTestingLogger(t, "pub"))

	inputs := []config.InputConfig{
		{Datastream: config.DatastreamConfig{Type: "logs", Namespace: "default"}},
	}
	if err := p.Configure(inputs); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if p.client == nil {
		t.Error("p.client is nil; empty-dataset input was not bound as result stream")
	}
}

func toMap(t *testing.T, s string) map[string]any {
	var m map[string]any
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
