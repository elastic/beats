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

package apmtest

import (
	"context"
	"fmt"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

// NewRecordingTracer returns a new RecordingTracer, containing a new
// Tracer using the RecorderTransport stored inside.
func NewRecordingTracer() *RecordingTracer {
	var result RecordingTracer
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		Transport: &result.RecorderTransport,
	})
	if err != nil {
		panic(err)
	}
	result.Tracer = tracer
	return &result
}

// RecordingTracer holds an apm.Tracer and transporttest.RecorderTransport.
type RecordingTracer struct {
	*apm.Tracer
	transporttest.RecorderTransport
}

// WithTransaction calls rt.WithTransactionOptions with a zero apm.TransactionOptions.
func (rt *RecordingTracer) WithTransaction(f func(ctx context.Context)) (model.Transaction, []model.Span, []model.Error) {
	return rt.WithTransactionOptions(apm.TransactionOptions{}, f)
}

// WithTransactionOptions starts a transaction with the given options,
// calls f with the transaction in the provided context, ends the transaction
// and flushes the tracer, and then returns the resulting events.
func (rt *RecordingTracer) WithTransactionOptions(opts apm.TransactionOptions, f func(ctx context.Context)) (model.Transaction, []model.Span, []model.Error) {
	tx := rt.StartTransactionOptions("name", "type", opts)
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	f(ctx)

	tx.End()
	rt.Flush(nil)
	payloads := rt.Payloads()
	if n := len(payloads.Transactions); n != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", n))
	}
	return payloads.Transactions[0], payloads.Spans, payloads.Errors
}
