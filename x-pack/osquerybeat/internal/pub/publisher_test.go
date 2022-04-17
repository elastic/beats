// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"github.com/menderesk/beats/v7/libbeat/beat/events"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/ecs"
)

func TestHitToEvent(t *testing.T) {

	const maxMask = 0b1111111

	type params struct {
		index, eventType, actionID, responseID string
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
		ev := hitToEvent(p.index, p.eventType, p.actionID, p.responseID, p.hit, p.ecsm, p.reqData)

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
