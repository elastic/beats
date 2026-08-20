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

package otelmap

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// BenchmarkFromMapstr measures encoding a mapstr.M into a pcommon.Map.
func BenchmarkFromMapstr(b *testing.B) {
	impls := []struct {
		name string
		fn   func(dst pcommon.Map, src mapstr.M) error
	}{
		{name: "default", fn: FromMapstr[mapstr.M]},
		{name: "legacy", fn: FromMapstrLegacy[mapstr.M]},
	}

	for _, impl := range impls {
		for _, tc := range BenchmarkCases() {
			b.Run(impl.name+"/"+tc.Name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					dst := pcommon.NewMap()
					if err := impl.fn(dst, tc.Src); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkToMapstr measures encoding a pcommon.Map into a new mapstr.M
func BenchmarkToMapstr(b *testing.B) {
	for _, tc := range BenchmarkCases() {
		src := pcommon.NewMap()
		if err := FromMapstr(src, tc.Src); err != nil {
			b.Fatalf("setup: %v", err)
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = ToMapstr(src)
			}
		})
	}
}
