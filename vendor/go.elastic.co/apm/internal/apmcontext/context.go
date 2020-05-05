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

package apmcontext

import "context"

var (
	// ContextWithSpan takes a context and span and returns a new context
	// from which the span can be extracted using SpanFromContext.
	//
	// ContextWithSpan is used by apm.ContextWithSpan. It is a
	// variable to allow other packages, such as apmot, to replace it
	// at package init time.
	ContextWithSpan = DefaultContextWithSpan

	// ContextWithTransaction takes a context and transaction and returns
	// a new context from which the transaction can be extracted using
	// TransactionFromContext.
	//
	// ContextWithTransaction is used by apm.ContextWithTransaction.
	// It is a variable to allow other packages, such as apmot, to replace
	// it at package init time.
	ContextWithTransaction = DefaultContextWithTransaction

	// SpanFromContext returns a span included in the context using
	// ContextWithSpan.
	//
	// SpanFromContext is used by apm.SpanFromContext. It is a
	// variable to allow other packages, such as apmot, to replace it
	// at package init time.
	SpanFromContext = DefaultSpanFromContext

	// TransactionFromContext returns a transaction included in the context
	// using ContextWithTransaction.
	//
	// TransactionFromContext is used by apm.TransactionFromContext.
	// It is a variable to allow other packages, such as apmot, to replace
	// it at package init time.
	TransactionFromContext = DefaultTransactionFromContext
)

type spanKey struct{}
type transactionKey struct{}

// DefaultContextWithSpan is the default value for ContextWithSpan.
func DefaultContextWithSpan(ctx context.Context, span interface{}) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

// DefaultContextWithTransaction is the default value for ContextWithTransaction.
func DefaultContextWithTransaction(ctx context.Context, tx interface{}) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

// DefaultSpanFromContext is the default value for SpanFromContext.
func DefaultSpanFromContext(ctx context.Context) interface{} {
	return ctx.Value(spanKey{})
}

// DefaultTransactionFromContext is the default value for TransactionFromContext.
func DefaultTransactionFromContext(ctx context.Context) interface{} {
	return ctx.Value(transactionKey{})
}
