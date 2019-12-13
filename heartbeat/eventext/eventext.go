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
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// MergeEventFields merges the given common.MapStr into the given Event's Fields.
func MergeEventFields(e *beat.Event, merge common.MapStr) {
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
		if event.Meta == nil {
			event.Meta = common.MapStr{}
		}
		event.Meta.Put(EventCancelledMetaKey, true)
	}
}

// IsEventCancelled checks for the marker left by CancelEvent.
func IsEventCancelled(event *beat.Event) bool {
	if event == nil || event.Meta == nil {
		return false
	}
	v, err := event.Meta.GetValue(EventCancelledMetaKey)
	return err == nil && v == true
}
