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

package javascript

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/tests/resources"
)

const (
	header = `function process(evt) {`
	footer = `}`
)

type testCase struct {
	name   string
	source string
	assert func(t testing.TB, evt *beat.Event, err error)
}

var eventV0Tests = []testCase{
	{
		name:   "Put",
		source: `evt.Put("hello", "world");`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			v, _ := evt.GetValue("hello")
			assert.Equal(t, "world", v)
		},
	},
	{
		name:   "Object Put Key",
		source: `evt.fields["hello"] = "world";`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			v, _ := evt.GetValue("hello")
			assert.Equal(t, "world", v)
		},
	},
	{
		name: "Get",
		source: `
			var ip = evt.Get("source.ip");

			if ("192.0.2.1" !== ip) {
				throw "failed to get IP";
			}`,
	},
	{
		name: "Get Object",
		source: `
			var source = evt.Get("source");

  			if ("192.0.2.1" !== source.ip) {
    			throw "failed to get IP";
  			}`,
	},
	{
		name: "Get Undefined Key",
		source: `
			var ip = evt.Get().source.ip;

  			if ("192.0.2.1" !== ip) {
    			throw "failed to get IP";
  			}`,
	},
	{
		name: "fields get key",
		source: `
			var ip = evt.fields.source.ip;

  			if ("192.0.2.1" !== ip) {
    			throw "failed to get IP";
  			}`,
	},
	{
		name:   "Delete",
		source: `if (!evt.Delete("source.ip")) { throw "delete failed"; }`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			ip, _ := evt.GetValue("source.ip")
			assert.Nil(t, ip)
		},
	},
	{
		name:   "Rename",
		source: `if (!evt.Rename("source", "destination")) { throw "rename failed"; }`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			ip, _ := evt.GetValue("destination.ip")
			assert.Equal(t, "192.0.2.1", ip)
		},
	},
	{
		name: "Get @metadata",
		source: `if (evt.Get("@metadata.pipeline") !== "beat-1.2.3-module") {
					throw "failed to get @metadata";
               }`,
	},
	{
		name:   "Put @metadata",
		source: `evt.Put("@metadata.foo", "bar");`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			assert.Equal(t, "bar", evt.Meta["foo"])
		},
	},
	{
		name:   "Delete @metadata",
		source: `evt.Delete("@metadata.pipeline");`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			assert.Nil(t, evt.Meta["pipeline"])
		},
	},
	{
		name:   "Cancel",
		source: `evt.Cancel();`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			assert.NoError(t, err)
			assert.Nil(t, evt)
		},
	},
	{
		name:   "Tag",
		source: `evt.Tag("foo"); evt.Tag("bar"); evt.Tag("foo");`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			if assert.NoError(t, err) {
				assert.Equal(t, []string{"foo", "bar"}, evt.Fields["tags"])
			}
		},
	},
	{
		name:   "AppendTo",
		source: `evt.AppendTo("source.ip", "10.0.0.1");`,
		assert: func(t testing.TB, evt *beat.Event, err error) {
			if assert.NoError(t, err) {
				srcIP, _ := evt.GetValue("source.ip")
				assert.Equal(t, []string{"192.0.2.1", "10.0.0.1"}, srcIP)
			}
		},
	},
}

func testEvent() *beat.Event {
	return &beat.Event{
		Meta: common.MapStr{
			"pipeline": "beat-1.2.3-module",
		},
		Fields: common.MapStr{
			"source": common.MapStr{
				"ip": "192.0.2.1",
			},
		},
	}
}

func TestBeatEventV0(t *testing.T) {
	for _, tc := range eventV0Tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := monitoring.NewRegistry()

			p, err := NewFromConfig(Config{Tag: tc.name, Source: header + tc.source + footer}, reg)
			if err != nil {
				t.Fatal(err)
			}

			evt, err := p.Run(testEvent())
			if tc.assert != nil {
				tc.assert(t, evt, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, evt)
			}

			// Validate that the processor's metrics exist.
			var found bool
			prefix := fmt.Sprintf("processor.javascript.%s.histogram.process_time", tc.name)
			reg.Do(monitoring.Full, func(name string, v interface{}) {
				if !found && strings.HasPrefix(name, prefix) {
					found = true
				}
			})
			assert.True(t, found, "metrics were not found in registry")
		})
	}

}

func BenchmarkBeatEventV0(b *testing.B) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(b)

	benchTest := func(tc testCase, timeout time.Duration) func(b *testing.B) {
		return func(b *testing.B) {
			p, err := NewFromConfig(Config{Source: header + tc.source + footer, Timeout: timeout}, nil)
			if err != nil {
				b.Fatal(err)
			}

			event := testEvent()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := p.Run(event)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	}
	for _, tc := range eventV0Tests {
		switch tc.name {
		case "Delete", "Rename":
			// Skip these tests for the benchmark because they affect the state
			// of the event in way that prevents them from being run more than
			// one time.
			continue
		}

		b.Run(tc.name, benchTest(tc, 0))
		b.Run("timeout_"+tc.name, benchTest(tc, 500*time.Millisecond))
	}
}
