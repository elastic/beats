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

// Package otelmap provides utilities for converting between beats and otel map types.
package otelmap

import (
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// ToMapstr converts a [pcommon.Map] to a [mapstr.M].
func ToMapstr(m pcommon.Map) mapstr.M {
	return m.AsRaw()
}

func ConvertNonPrimitive(m mapstr.M) {
	for key, val := range m {
		switch x := val.(type) {
		case mapstr.M:
			ConvertNonPrimitive(x)
		case time.Time:
			m[key] = x.UTC().Format("2006-01-02T15:04:05.000Z")
		case []time.Time:
			s := make([]any, 0, len(x))
			for _, i := range x {
				s = append(s, i.UTC().Format("2006-01-02T15:04:05.000Z"))
			}
			m[key] = s
		case []mapstr.M:
			s := make([]any, len(x))
			for i, val := range x {
				ConvertNonPrimitive(val)
				s[i] = val
			}
			m[key] = s
		case []bool:
			m[key] = convertSlice(x)
		case []string:
			m[key] = convertSlice(x)
		case []float32:
			m[key] = convertSlice(x)
		case []float64:
			m[key] = convertSlice(x)
		case []int:
			m[key] = convertSlice(x)
		case []int16:
			m[key] = convertSlice(x)
		case []int32:
			m[key] = convertSlice(x)
		case []int64:
			m[key] = convertSlice(x)
		case []uint:
			m[key] = convertSlice(x)
		case []uint16:
			m[key] = convertSlice(x)
		case []uint32:
			m[key] = convertSlice(x)
		case []uint64:
			m[key] = convertSlice(x)
		}

	}
}

func convertSlice[T any](slice []T) []any {
	s := make([]any, len(slice))
	for i, val := range slice {
		s[i] = val
	}
	return s
}
