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

package dissect

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchmarkProcessorRunOverwriteKeys benchmarks the full processor.Run() path
// with OverwriteKeys=true, which is the common production configuration.
func BenchmarkProcessorRunOverwriteKeys(b *testing.B) {
	c, err := cfg.NewConfigFrom(map[string]interface{}{
		"tokenizer":      "%{timestamp} %{log_level}  [%{logger}] %{source} %{message}",
		"field":          "message",
		"target_prefix":  "dissect",
		"overwrite_keys": true,
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := NewProcessor(c, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{
			Fields: mapstr.M{
				"message": "2025-03-07T11:06:39.123456789Z INFO  [application] app/server.go:142 Request processed successfully",
				"agent":   mapstr.M{"name": "test", "version": "8.17.0"},
				"host":    mapstr.M{"name": "myhost", "os": mapstr.M{"type": "linux"}},
			},
		}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
