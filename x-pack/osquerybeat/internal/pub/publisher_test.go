// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
)

func TestHitToEvent(t *testing.T) {

	const maxMask = 0b1111111

	type params struct {
		index, eventType, actionID, responseID string
		meta                                   map[string]interface{}
		hit                                    map[string]interface{}
		ecsm                                   ecs.Mapping
		reqData                                interface{}
	}

	genParams := func(mask int) (p params) {
		if mask>>6&1 > 0 {
			p.index = "logs-osquery_manager.result-default"
		}
		if mask>>5&1 > 0 {
			p.eventType = "osquery_manager"
		}
		if mask>>4&1 > 0 {
			p.actionID = "uptime"
		}
		if mask>>3&1 > 0 {
			p.responseID = uuid.Must(uuid.NewV4()).String()
		}
		if mask>>2&1 > 0 {
			p.hit = map[string]interface{}{
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
			p.reqData = map[string]interface{}{
				"query": "select * from uptime",
			}
		}
		return p
	}

	for i := 0; i < maxMask; i++ {
		p := genParams(i)
		ev := hitToEvent(p.index, p.eventType, p.actionID, p.responseID, p.meta, p.hit, p.ecsm, p.reqData)

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

		diff = cmp.Diff(p.actionID, ev.Fields["action_id"])
		if diff != "" {
			t.Error(diff)
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
		req, res map[string]interface{}
		want     map[string]interface{}
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
			// "agent_id": "bf3d6036-2260-4bbf-94a3-5ccce0d75d9e",
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
			// "agent_id": "bf3d6036-2260-4bbf-94a3-5ccce0d75d9e",
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
	}

	for _, tc := range tests {
		got := actionResultToEvent(tc.req, tc.res)
		diff := cmp.Diff(tc.want, got)
		if diff != "" {
			t.Error(diff)
		}
	}
}

func toMap(t *testing.T, s string) map[string]interface{} {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
