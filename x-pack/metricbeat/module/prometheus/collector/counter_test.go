// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"testing"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
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
				if got, _ := tt.counterCache.RateUint64(tt.counterName, val); got != want {
					t.Errorf("counterCache.RateUint64() = %v, want %v", got, want)
				}
			}
			for i, val := range tt.valuesFloat64 {
				want := tt.expectedFloat64[i]
				if got, _ := tt.counterCache.RateFloat64(tt.counterName, val); got != want {
					t.Errorf("counterCache.RateFloat64() = %v, want %v", got, want)
				}
			}
		})
	}
}
