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

// TracerStats holds statistics for a Tracer.
type TracerStats struct {
	Errors              TracerStatsErrors
	ErrorsSent          uint64
	ErrorsDropped       uint64
	TransactionsSent    uint64
	TransactionsDropped uint64
	SpansSent           uint64
	SpansDropped        uint64
}

// TracerStatsErrors holds error statistics for a Tracer.
type TracerStatsErrors struct {
	SetContext uint64
	SendStream uint64
}

func (s TracerStats) isZero() bool {
	return s == TracerStats{}
}

// accumulate updates the stats by accumulating them with
// the values in rhs.
func (s *TracerStats) accumulate(rhs TracerStats) {
	s.Errors.SetContext += rhs.Errors.SetContext
	s.Errors.SendStream += rhs.Errors.SendStream
	s.ErrorsSent += rhs.ErrorsSent
	s.ErrorsDropped += rhs.ErrorsDropped
	s.SpansSent += rhs.SpansSent
	s.SpansDropped += rhs.SpansDropped
	s.TransactionsSent += rhs.TransactionsSent
	s.TransactionsDropped += rhs.TransactionsDropped
}
