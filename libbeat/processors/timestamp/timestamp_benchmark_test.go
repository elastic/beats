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

package timestamp

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchmarkTimestampSingleLayout measures the common case: one layout that
// matches every event. This is the hot path in most filebeat deployments.
func BenchmarkTimestampSingleLayout(b *testing.B) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = []string{time.RFC3339Nano}

	p, err := newFromConfig(c, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	tsStr := time.Date(2025, 3, 7, 11, 6, 39, 123456789, time.UTC).Format(time.RFC3339Nano)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{Fields: mapstr.M{"ts": tsStr}}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTimestampMultipleLayouts measures the case where multiple layouts
// are configured and the matching layout is the last one tried.
func BenchmarkTimestampMultipleLayouts(b *testing.B) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = []string{time.ANSIC, time.RFC822, time.RFC3339Nano}

	p, err := newFromConfig(c, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	// Use RFC3339Nano format so the first two layouts fail.
	tsStr := time.Date(2025, 3, 7, 11, 6, 39, 123456789, time.UTC).Format(time.RFC3339Nano)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{Fields: mapstr.M{"ts": tsStr}}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
