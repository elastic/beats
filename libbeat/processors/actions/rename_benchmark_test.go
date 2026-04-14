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

package actions

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchmarkRenameSingleField benchmarks a single-field rename with different
// top-level keys (e.g. message → event.original), the common agent pattern.
func BenchmarkRenameSingleField(b *testing.B) {
	c, err := conf.NewConfigFrom(map[string]interface{}{
		"fields": []map[string]interface{}{
			{"from": "message", "to": "event.original"},
		},
		"fail_on_error": true,
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := NewRenameFields(c, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{
			Fields: mapstr.M{
				"message": "test log line with some content",
				"agent":   mapstr.M{"name": "test", "version": "8.17.0"},
				"host":    mapstr.M{"name": "myhost", "os": mapstr.M{"type": "linux"}},
				"cloud":   mapstr.M{"provider": "gcp", "region": "us-central1"},
			},
		}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
