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

package outputs

import (
	"sync"

	"github.com/rcrowley/go-metrics"
)

// lazySample is a uniform sample whose reservoir is only allocated on the
// first Update. A uniform sample reserves its full size up front, and a Beat
// that never reports send latency (every Beat running as an OTel receiver)
// would otherwise hold that memory for nothing.
type lazySample struct {
	reservoirSize int

	mu     sync.Mutex
	sample metrics.Sample // nil until the first Update
}

func newLazySample(reservoirSize int) *lazySample {
	return &lazySample{reservoirSize: reservoirSize}
}

// get returns the backing sample, or nil if nothing was recorded yet.
func (s *lazySample) get() metrics.Sample {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sample
}

func (s *lazySample) Update(v int64) {
	s.mu.Lock()
	if s.sample == nil {
		s.sample = metrics.NewUniformSample(s.reservoirSize)
	}
	sample := s.sample
	s.mu.Unlock()
	sample.Update(v)
}

func (s *lazySample) Clear() {
	if sample := s.get(); sample != nil {
		sample.Clear()
	}
}

func (s *lazySample) Snapshot() metrics.Sample {
	if sample := s.get(); sample != nil {
		return sample.Snapshot()
	}
	return metrics.NewSampleSnapshot(0, nil)
}

func (s *lazySample) Size() int { return s.snapshot().Size() }

func (s *lazySample) Count() int64 { return s.snapshot().Count() }

func (s *lazySample) Max() int64 { return s.snapshot().Max() }

func (s *lazySample) Min() int64 { return s.snapshot().Min() }

func (s *lazySample) Sum() int64 { return s.snapshot().Sum() }

func (s *lazySample) Mean() float64 { return s.snapshot().Mean() }

func (s *lazySample) StdDev() float64 { return s.snapshot().StdDev() }

func (s *lazySample) Variance() float64 { return s.snapshot().Variance() }

func (s *lazySample) Values() []int64 { return s.snapshot().Values() }

func (s *lazySample) Percentile(p float64) float64 { return s.snapshot().Percentile(p) }

func (s *lazySample) Percentiles(ps []float64) []float64 { return s.snapshot().Percentiles(ps) }

// snapshot is the Sample the read-only accessors delegate to.
func (s *lazySample) snapshot() metrics.Sample {
	if sample := s.get(); sample != nil {
		return sample
	}
	return emptySample
}

var emptySample = metrics.NewSampleSnapshot(0, nil)
