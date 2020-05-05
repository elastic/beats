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

package apm

import (
	"context"

	"go.elastic.co/apm/internal/apmcontext"
)

// ContextWithSpan returns a copy of parent in which the given span
// is stored, associated with the key ContextSpanKey.
func ContextWithSpan(parent context.Context, s *Span) context.Context {
	return apmcontext.ContextWithSpan(parent, s)
}

// ContextWithTransaction returns a copy of parent in which the given
// transaction is stored, associated with the key ContextTransactionKey.
func ContextWithTransaction(parent context.Context, t *Transaction) context.Context {
	return apmcontext.ContextWithTransaction(parent, t)
}

// SpanFromContext returns the current Span in context, if any. The span must
// have been added to the context previously using ContextWithSpan, or the
// top-level StartSpan function.
func SpanFromContext(ctx context.Context) *Span {
	value, _ := apmcontext.SpanFromContext(ctx).(*Span)
	return value
}

// TransactionFromContext returns the current Transaction in context, if any.
// The transaction must have been added to the context previously using
// ContextWithTransaction.
func TransactionFromContext(ctx context.Context) *Transaction {
	value, _ := apmcontext.TransactionFromContext(ctx).(*Transaction)
	return value
}

// DetachedContext returns a new context detached from the lifetime
// of ctx, but which still returns the values of ctx.
//
// DetachedContext can be used to maintain the trace context required
// to correlate events, but where the operation is "fire-and-forget",
// and should not be affected by the deadline or cancellation of ctx.
func DetachedContext(ctx context.Context) context.Context {
	return &detachedContext{Context: context.Background(), orig: ctx}
}

type detachedContext struct {
	context.Context
	orig context.Context
}

// Value returns c.orig.Value(key).
func (c *detachedContext) Value(key interface{}) interface{} {
	return c.orig.Value(key)
}

// StartSpan is equivalent to calling StartSpanOptions with a zero SpanOptions struct.
func StartSpan(ctx context.Context, name, spanType string) (*Span, context.Context) {
	return StartSpanOptions(ctx, name, spanType, SpanOptions{})
}

// StartSpanOptions starts and returns a new Span within the sampled transaction
// and parent span in the context, if any. If the span isn't dropped, it will be
// stored in the resulting context.
//
// If opts.Parent is non-zero, its value will be used in preference to any parent
// span in ctx.
//
// StartSpanOptions always returns a non-nil Span. Its End method must be called
// when the span completes.
func StartSpanOptions(ctx context.Context, name, spanType string, opts SpanOptions) (*Span, context.Context) {
	var span *Span
	if opts.parent = SpanFromContext(ctx); opts.parent != nil {
		if opts.parent.tx == nil && opts.parent.tracer != nil {
			span = opts.parent.tracer.StartSpan(name, spanType, opts.parent.transactionID, opts)
		} else {
			span = opts.parent.tx.StartSpanOptions(name, spanType, opts)
		}
	} else {
		tx := TransactionFromContext(ctx)
		span = tx.StartSpanOptions(name, spanType, opts)
	}
	if !span.Dropped() {
		ctx = ContextWithSpan(ctx, span)
	}
	return span, ctx
}

// CaptureError returns a new Error related to the sampled transaction
// and span present in the context, if any, and sets its exception info
// from err. The Error.Handled field will be set to true, and a stacktrace
// set either from err, or from the caller.
//
// If the provided error is nil, then CaptureError will also return nil;
// otherwise a non-nil Error will always be returned. If there is no
// transaction or span in the context, then the returned Error's Send
// method will have no effect.
func CaptureError(ctx context.Context, err error) *Error {
	if err == nil {
		return nil
	}
	if span := SpanFromContext(ctx); span != nil {
		if span.tracer == nil {
			return &Error{cause: err, err: err.Error()}
		}
		e := span.tracer.NewError(err)
		e.Handled = true
		e.SetSpan(span)
		return e
	} else if tx := TransactionFromContext(ctx); tx != nil {
		if tx.tracer == nil {
			return &Error{cause: err, err: err.Error()}
		}
		e := tx.tracer.NewError(err)
		e.Handled = true
		e.SetTransaction(tx)
		return e
	} else {
		return &Error{cause: err, err: err.Error()}
	}
}
