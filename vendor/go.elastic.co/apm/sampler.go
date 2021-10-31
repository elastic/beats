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

package apm // import "go.elastic.co/apm"

import (
	"encoding/binary"
	"math"
	"math/big"

	"github.com/pkg/errors"
)

// Sampler provides a means of sampling transactions.
type Sampler interface {
	// Sample indicates whether or not a transaction
	// should be sampled. This method will be invoked
	// by calls to Tracer.StartTransaction for the root
	// of a trace, so it must be goroutine-safe, and
	// should avoid synchronization as far as possible.
	Sample(TraceContext) bool
}

// ExtendedSampler may be implemented by Samplers, providing
// a method for sampling and returning an extended SampleResult.
//
// TODO(axw) in v2.0.0, replace the Sampler interface with this.
type ExtendedSampler interface {
	// SampleExtended indicates whether or not a transaction
	// should be sampled, and the sampling rate in effect at
	// the time. This method will be invoked by calls to
	// Tracer.StartTransaction for the root of a trace, so it
	// must be goroutine-safe, and should avoid synchronization
	// as far as possible.
	SampleExtended(SampleParams) SampleResult
}

// SampleParams holds parameters for SampleExtended.
type SampleParams struct {
	// TraceContext holds the newly-generated TraceContext
	// for the root transaction which is being sampled.
	TraceContext TraceContext
}

// SampleResult holds information about a sampling decision.
type SampleResult struct {
	// Sampled holds the sampling decision.
	Sampled bool

	// SampleRate holds the sample rate in effect at the
	// time of the sampling decision. This is used for
	// propagating the value downstream, and for inclusion
	// in events sent to APM Server.
	//
	// The sample rate will be rounded to 4 decimal places
	// half away from zero, except if it is in the interval
	// (0, 0.0001], in which case it is set to 0.0001. The
	// Sampler implementation should also internally apply
	// this logic to ensure consistency.
	SampleRate float64
}

// NewRatioSampler returns a new Sampler with the given ratio
//
// A ratio of 1.0 samples 100% of transactions, a ratio of 0.5
// samples ~50%, and so on. If the ratio provided does not lie
// within the range [0,1.0], NewRatioSampler will panic.
//
// Sampling rate is rounded to 4 digits half away from zero,
// except if it is in the interval (0, 0.0001], in which case
// is set to 0.0001.
//
// The returned Sampler bases its decision on the value of the
// transaction ID, so there is no synchronization involved.
func NewRatioSampler(r float64) Sampler {
	if r < 0 || r > 1.0 {
		panic(errors.Errorf("ratio %v out of range [0,1.0]", r))
	}
	r = roundSampleRate(r)
	var x big.Float
	x.SetUint64(math.MaxUint64)
	x.Mul(&x, big.NewFloat(r))
	ceil, _ := x.Uint64()
	return ratioSampler{r, ceil}
}

type ratioSampler struct {
	ratio float64
	ceil  uint64
}

// Sample samples the transaction according to the configured
// ratio and pseudo-random source.
func (s ratioSampler) Sample(c TraceContext) bool {
	return s.SampleExtended(SampleParams{TraceContext: c}).Sampled
}

// SampleExtended samples the transaction according to the configured
// ratio and pseudo-random source.
func (s ratioSampler) SampleExtended(args SampleParams) SampleResult {
	v := binary.BigEndian.Uint64(args.TraceContext.Span[:])
	result := SampleResult{
		Sampled:    v > 0 && v-1 < s.ceil,
		SampleRate: s.ratio,
	}
	return result
}

// roundSampleRate rounds r to 4 decimal places half away from zero,
// with the exception of values > 0 and < 0.0001, which are set to 0.0001.
func roundSampleRate(r float64) float64 {
	if r > 0 && r < 0.0001 {
		r = 0.0001
	}
	return round(r*10000) / 10000
}
