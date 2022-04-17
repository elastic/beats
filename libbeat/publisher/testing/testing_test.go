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

package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

var cnt = 0

func testEvent() beat.Event {
	event := beat.Event{
		Fields: common.MapStr{
			"message": "test",
			"idx":     cnt,
		},
	}
	cnt++
	return event
}

// Test that ChanClient writes an event to its Channel.
func TestChanClientPublishEvent(t *testing.T) {
	cc := NewChanClient(1)
	e1 := testEvent()
	cc.Publish(e1)
	assert.Equal(t, e1, cc.ReceiveEvent())
}

// Test that ChanClient write events to its Channel.
func TestChanClientPublishEvents(t *testing.T) {
	cc := NewChanClient(1)

	e1, e2 := testEvent(), testEvent()
	go cc.PublishAll([]beat.Event{e1, e2})
	assert.Equal(t, e1, cc.ReceiveEvent())
	assert.Equal(t, e2, cc.ReceiveEvent())
}
