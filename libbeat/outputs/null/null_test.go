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

package null

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/format"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/publisher"
)

func TestNullOutput(t *testing.T) {
	tests := []struct {
		title  string
		codec  codec.Codec
		events []beat.Event
	}{
		{
			"single json event (pretty=false)",
			json.New(false, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
		},
		{
			"single json event (pretty=true)",
			json.New(true, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
		},
		{
			"event with custom format string",
			format.New(fmtstr.MustCompileEvent("%{[event]}")),
			[]beat.Event{
				{Fields: event("event", "myevent")},
			},
		},
		{
			"event with no codec",
			nil,
			[]beat.Event{
				{Fields: event("event", "myevent")},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			batch := outest.NewBatch(test.events...)
			run(test.codec, batch)

			// check batch correctly signalled
			if !assert.Len(t, batch.Signals, 1) {
				return
			}
			assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		})
	}
}

func newNull(observer outputs.Observer, codec codec.Codec) (*null, error) {
	return &null{codec: codec, observer: observer}, nil
}

func run(codec codec.Codec, batches ...publisher.Batch) {
	c, _ := newNull(outputs.NewNilObserver(), codec)
	for _, b := range batches {
		c.Publish(b)
	}

}

func event(k, v string) common.MapStr {
	return common.MapStr{k: v}
}
