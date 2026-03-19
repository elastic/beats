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
	"fmt"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BenchmarkAddFieldsMultipleSingleProcessor benchmarks a single add_fields_multiple
// processor with N entries.
func BenchmarkAddFieldsMultipleSingleProcessor(b *testing.B) {
	fieldCounts := []int{5, 10, 20, 50, 100}

	for _, n := range fieldCounts {
		b.Run(fmt.Sprintf("%d_fields", n), func(b *testing.B) {
			entries := make([]string, n)
			for i := 0; i < n; i++ {
				entries[i] = fmt.Sprintf("{fields: {field%d: value%d}}", i, i)
			}
			configYAML := fmt.Sprintf("add_fields_multiple: [%s]", strings.Join(entries, ", "))

			config, err := conf.NewConfigWithYAML([]byte(configYAML), "test")
			if err != nil {
				b.Fatal(err)
			}

			procCfg, err := config.Child("add_fields_multiple", -1)
			if err != nil {
				b.Fatal(err)
			}

			proc, err := addfields.CreateAddFieldsMultiple(procCfg, logptest.NewTestingLogger(b, ""))
			if err != nil {
				b.Fatal(err)
			}

			event := &beat.Event{Fields: mapstr.M{}}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ev := event.Clone()
				_, _ = proc.Run(ev)
			}
		})
	}
}

// BenchmarkAddFieldsMultipleProcessors benchmarks N separate add_fields processors,
// each adding a single field, run in sequence.
func BenchmarkAddFieldsMultipleProcessors(b *testing.B) {
	fieldCounts := []int{5, 10, 20, 50, 100}

	for _, n := range fieldCounts {
		b.Run(fmt.Sprintf("%d_fields", n), func(b *testing.B) {
			procs := make([]beat.Processor, n)
			for i := range n {
				fields := mapstr.M{
					fmt.Sprintf("field%d", i): fmt.Sprintf("value%d", i),
				}
				procs[i] = addfields.MakeFieldsProcessor(addfields.FieldsKey, fields, true)
			}

			event := &beat.Event{Fields: mapstr.M{}}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ev := event.Clone()
				for _, p := range procs {
					var err error
					ev, err = p.Run(ev)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
