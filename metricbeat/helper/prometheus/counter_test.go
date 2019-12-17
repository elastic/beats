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

package prometheus

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

func Test_CounterCache(t *testing.T) {
	type fields struct {
		ints    *common.Cache
		floats  *common.Cache
		timeout time.Duration
	}

	tests := []struct {
		name            string
		counterCache    CounterCache
		counterName     string
		valuesUint64    []uint64
		expectedUin64   []uint64
		valuesFloat64   []float64
		expectedFloat64 []float64
	}{
		{
			name:            "rates are calculated",
			counterCache:    NewCounterCache(1 * time.Second),
			counterName:     "test_counter",
			valuesUint64:    []uint64{10, 14, 17, 17, 28},
			expectedUin64:   []uint64{0, 4, 3, 0, 11},
			valuesFloat64:   []float64{1.0, 101.0, 102.0, 102.0, 1034.0},
			expectedFloat64: []float64{0.0, 100.0, 1.0, 0.0, 932.0},
		},
		{
			name:            "counter reset",
			counterCache:    NewCounterCache(1 * time.Second),
			counterName:     "test_counter",
			valuesUint64:    []uint64{10, 14, 17, 1, 3},
			expectedUin64:   []uint64{0, 4, 3, 0, 2},
			valuesFloat64:   []float64{1.0, 101.0, 2.0, 13.0},
			expectedFloat64: []float64{0.0, 100.0, 0.0, 11.0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, val := range tt.valuesUint64 {
				want := tt.expectedUin64[i]
				if got := tt.counterCache.RateUint64(tt.counterName, val); got != want {
					t.Errorf("counterCache.RateUint64() = %v, want %v", got, want)
				}
			}
			for i, val := range tt.valuesFloat64 {
				want := tt.expectedFloat64[i]
				if got := tt.counterCache.RateFloat64(tt.counterName, val); got != want {
					t.Errorf("counterCache.RateFloat64() = %v, want %v", got, want)
				}
			}
		})
	}
}
