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

package eventext

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// MergeEventFields merges the given mapstr.M into the given Event's Fields.
func MergeEventFields(e *beat.Event, merge mapstr.M) {
	if e.Fields != nil {
		e.Fields.DeepUpdate(merge.Clone())
	} else {
		e.Fields = merge.Clone()
	}
}

// EventCancelledMetaKey is the path to the @metadata key marking an event as cancelled.
const EventCancelledMetaKey = "__hb_evt_cancel__"

// CancelEvent marks the event as cancelled. Downstream consumers of it should not emit nor output this event.
func CancelEvent(event *beat.Event) {
	if event != nil {
		SetMeta(event, EventCancelledMetaKey, true)
	}
}

func SetMeta(event *beat.Event, k string, v interface{}) {
	if event.Meta == nil {
		event.Meta = mapstr.M{}
	}
	_, _ = event.Meta.Put(k, v)
}

// IsEventCancelled checks for the marker left by CancelEvent.
func IsEventCancelled(event *beat.Event) bool {
	if event == nil || event.Meta == nil {
		return false
	}
	v, err := event.Meta.GetValue(EventCancelledMetaKey)
	return err == nil && v == true
}
