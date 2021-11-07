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
	cryptorand "crypto/rand"
	"encoding/binary"
	"strings"
	"sync"
	"time"

	"go.elastic.co/apm/stacktrace"
)

// droppedSpanDataPool holds *SpanData which are used when the span is created
// for a nil or non-sampled trace context, without a transaction reference.
//
// Spans started with a non-nil transaction, even if it is non-sampled, are
// always created with the transaction's tracer span pool.
var droppedSpanDataPool sync.Pool

// StartSpan starts and returns a new Span within the transaction,
// with the specified name, type, and optional parent span, and
// with the start time set to the current time.
//
// StartSpan always returns a non-nil Span, with a non-nil SpanData
// field. Its End method must be called when the span completes.
//
// If the span type contains two dots, they are assumed to separate
// the span type, subtype, and action; a single dot separates span
// type and subtype, and the action will not be set.
//
// StartSpan is equivalent to calling StartSpanOptions with
// SpanOptions.Parent set to the trace context of parent if
// parent is non-nil.
func (tx *Transaction) StartSpan(name, spanType string, parent *Span) *Span {
	return tx.StartSpanOptions(name, spanType, SpanOptions{
		parent: parent,
	})
}

// StartSpanOptions starts and returns a new Span within the transaction,
// with the specified name, type, and options.
//
// StartSpan always returns a non-nil Span. Its End method must be called
// when the span completes.
//
// If the span type contains two dots, they are assumed to separate the
// span type, subtype, and action; a single dot separates span type and
// subtype, and the action will not be set.
func (tx *Transaction) StartSpanOptions(name, spanType string, opts SpanOptions) *Span {
	if tx == nil {
		return newDroppedSpan()
	}

	if opts.Parent == (TraceContext{}) {
		if opts.parent != nil {
			opts.Parent = opts.parent.TraceContext()
		} else {
			opts.Parent = tx.traceContext
		}
	}
	transactionID := tx.traceContext.Span

	// Prevent tx from being ended while we're starting a span.
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	if tx.ended() {
		return tx.tracer.StartSpan(name, spanType, transactionID, opts)
	}

	// Calculate the span time relative to the transaction timestamp so
	// that wall-clock adjustments occurring after the transaction start
	// don't affect the span timestamp.
	if opts.Start.IsZero() {
		opts.Start = tx.timestamp.Add(time.Since(tx.timestamp))
	} else {
		opts.Start = tx.timestamp.Add(opts.Start.Sub(tx.timestamp))
	}
	span := tx.tracer.startSpan(name, spanType, transactionID, opts)
	span.tx = tx
	span.parent = opts.parent

	// Guard access to spansCreated, spansDropped, rand, and childrenTimer.
	tx.TransactionData.mu.Lock()
	defer tx.TransactionData.mu.Unlock()
	if !span.traceContext.Options.Recorded() {
		span.tracer = nil // span is dropped
	} else if tx.maxSpans >= 0 && tx.spansCreated >= tx.maxSpans {
		span.tracer = nil // span is dropped
		tx.spansDropped++
	} else {
		if opts.SpanID.Validate() == nil {
			span.traceContext.Span = opts.SpanID
		} else {
			binary.LittleEndian.PutUint64(span.traceContext.Span[:], tx.rand.Uint64())
		}
		span.stackFramesMinDuration = tx.spanFramesMinDuration
		span.stackTraceLimit = tx.stackTraceLimit
		tx.spansCreated++
	}

	if tx.breakdownMetricsEnabled {
		if span.parent != nil {
			span.parent.mu.Lock()
			defer span.parent.mu.Unlock()
			if !span.parent.ended() {
				span.parent.childrenTimer.childStarted(span.timestamp)
			}
		} else {
			tx.childrenTimer.childStarted(span.timestamp)
		}
	}
	return span
}

// StartSpan returns a new Span with the specified name, type, transaction ID,
// and options. The parent transaction context and transaction IDs must have
// valid, non-zero values, or else the span will be dropped.
//
// In most cases, you should use Transaction.StartSpan or Transaction.StartSpanOptions.
// This method is provided for corner-cases, such as starting a span after the
// containing transaction's End method has been called. Spans created in this
// way will not have the "max spans" configuration applied, nor will they be
// considered in any transaction's span count.
func (t *Tracer) StartSpan(name, spanType string, transactionID SpanID, opts SpanOptions) *Span {
	if opts.Parent.Trace.Validate() != nil || opts.Parent.Span.Validate() != nil || transactionID.Validate() != nil {
		return newDroppedSpan()
	}
	if !opts.Parent.Options.Recorded() {
		return newDroppedSpan()
	}
	var spanID SpanID
	if opts.SpanID.Validate() == nil {
		spanID = opts.SpanID
	} else {
		if _, err := cryptorand.Read(spanID[:]); err != nil {
			return newDroppedSpan()
		}
	}
	if opts.Start.IsZero() {
		opts.Start = time.Now()
	}
	span := t.startSpan(name, spanType, transactionID, opts)
	span.traceContext.Span = spanID

	instrumentationConfig := t.instrumentationConfig()
	span.stackFramesMinDuration = instrumentationConfig.spanFramesMinDuration
	span.stackTraceLimit = instrumentationConfig.stackTraceLimit

	return span
}

// SpanOptions holds options for Transaction.StartSpanOptions and Tracer.StartSpan.
type SpanOptions struct {
	// Parent, if non-zero, holds the trace context of the parent span.
	Parent TraceContext

	// SpanID holds the ID to assign to the span. If this is zero, a new ID
	// will be generated and used instead.
	SpanID SpanID

	// parent, if non-nil, holds the parent span.
	//
	// This is only used if Parent is zero, and is only available to internal
	// callers of Transaction.StartSpanOptions.
	parent *Span

	// Start is the start time of the span. If this has the zero value,
	// time.Now() will be used instead.
	//
	// When a span is created using Transaction.StartSpanOptions, the
	// span timestamp is internally calculated relative to the transaction
	// timestamp.
	//
	// When Tracer.StartSpan is used, this timestamp should be pre-calculated
	// as relative from the transaction start time, i.e. by calculating the
	// time elapsed since the transaction started, and adding that to the
	// transaction timestamp. Calculating the timstamp in this way will ensure
	// monotonicity of events within a transaction.
	Start time.Time
}

func (t *Tracer) startSpan(name, spanType string, transactionID SpanID, opts SpanOptions) *Span {
	sd, _ := t.spanDataPool.Get().(*SpanData)
	if sd == nil {
		sd = &SpanData{Duration: -1}
	}
	span := &Span{tracer: t, SpanData: sd}
	span.Name = name
	span.traceContext = opts.Parent
	span.parentID = opts.Parent.Span
	span.transactionID = transactionID
	span.timestamp = opts.Start
	span.Type = spanType
	if dot := strings.IndexRune(spanType, '.'); dot != -1 {
		span.Type = spanType[:dot]
		span.Subtype = spanType[dot+1:]
		if dot := strings.IndexRune(span.Subtype, '.'); dot != -1 {
			span.Subtype, span.Action = span.Subtype[:dot], span.Subtype[dot+1:]
		}
	}
	return span
}

// newDropped returns a new Span with a non-nil SpanData.
func newDroppedSpan() *Span {
	span, _ := droppedSpanDataPool.Get().(*Span)
	if span == nil {
		span = &Span{SpanData: &SpanData{}}
	}
	return span
}

// Span describes an operation within a transaction.
type Span struct {
	tracer        *Tracer // nil if span is dropped
	tx            *Transaction
	parent        *Span
	traceContext  TraceContext
	transactionID SpanID

	mu sync.RWMutex

	// SpanData holds the span data. This field is set to nil when
	// the span's End method is called.
	*SpanData
}

// TraceContext returns the span's TraceContext.
func (s *Span) TraceContext() TraceContext {
	if s == nil {
		return TraceContext{}
	}
	return s.traceContext
}

// SetStacktrace sets the stacktrace for the span,
// skipping the first skip number of frames,
// excluding the SetStacktrace function.
func (s *Span) SetStacktrace(skip int) {
	if s == nil || s.dropped() {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ended() {
		return
	}
	s.SpanData.setStacktrace(skip + 1)
}

// Dropped indicates whether or not the span is dropped, meaning it will not
// be included in any transaction. Spans are dropped by Transaction.StartSpan
// if the transaction is nil, non-sampled, or the transaction's max spans
// limit has been reached.
//
// Dropped may be used to avoid any expensive computation required to set
// the span's context.
func (s *Span) Dropped() bool {
	return s == nil || s.dropped()
}

func (s *Span) dropped() bool {
	return s.tracer == nil
}

// End marks the s as being complete; s must not be used after this.
//
// If s.Duration has not been set, End will set it to the elapsed time
// since the span's start time.
func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended() {
		return
	}
	if s.Duration < 0 {
		s.Duration = time.Since(s.timestamp)
	}
	if s.Outcome == "" {
		s.Outcome = s.Context.outcome()
	}
	if s.dropped() {
		if s.tx == nil {
			droppedSpanDataPool.Put(s.SpanData)
		} else {
			s.reportSelfTime()
			s.reset(s.tx.tracer)
		}
		s.SpanData = nil
		return
	}
	if len(s.stacktrace) == 0 && s.Duration >= s.stackFramesMinDuration {
		s.setStacktrace(1)
	}
	if s.tx != nil {
		s.reportSelfTime()
	}
	s.enqueue()
	s.SpanData = nil
}

// reportSelfTime reports the span's self-time to its transaction, and informs
// the parent that it has ended in order for the parent to later calculate its
// own self-time.
//
// This must only be called from Span.End, with s.mu.Lock held for writing and
// s.Duration set.
func (s *Span) reportSelfTime() {
	endTime := s.timestamp.Add(s.Duration)

	// TODO(axw) try to find a way to not lock the transaction when
	// ending every span. We already lock them when starting spans.
	s.tx.mu.RLock()
	defer s.tx.mu.RUnlock()
	if s.tx.ended() || !s.tx.breakdownMetricsEnabled {
		return
	}

	s.tx.TransactionData.mu.Lock()
	defer s.tx.TransactionData.mu.Unlock()
	if s.parent != nil {
		s.parent.mu.Lock()
		if !s.parent.ended() {
			s.parent.childrenTimer.childEnded(endTime)
		}
		s.parent.mu.Unlock()
	} else {
		s.tx.childrenTimer.childEnded(endTime)
	}
	s.tx.spanTimings.add(s.Type, s.Subtype, s.Duration-s.childrenTimer.finalDuration(endTime))
}

func (s *Span) enqueue() {
	event := tracerEvent{eventType: spanEvent}
	event.span.Span = s
	event.span.SpanData = s.SpanData
	select {
	case s.tracer.events <- event:
	default:
		// Enqueuing a span should never block.
		s.tracer.statsMu.Lock()
		s.tracer.stats.SpansDropped++
		s.tracer.statsMu.Unlock()
		s.reset(s.tracer)
	}
}

func (s *Span) ended() bool {
	return s.SpanData == nil
}

// SpanData holds the details for a span, and is embedded inside Span.
// When a span is ended or discarded, its SpanData field will be set
// to nil.
type SpanData struct {
	parentID               SpanID
	stackFramesMinDuration time.Duration
	stackTraceLimit        int
	timestamp              time.Time
	childrenTimer          childrenTimer

	// Name holds the span name, initialized with the value passed to StartSpan.
	Name string

	// Type holds the overarching span type, such as "db", and will be initialized
	// with the value passed to StartSpan.
	Type string

	// Subtype holds the span subtype, such as "mysql". This will initially be empty,
	// and can be set after starting the span.
	Subtype string

	// Action holds the span action, such as "query". This will initially be empty,
	// and can be set after starting the span.
	Action string

	// Duration holds the span duration, initialized to -1.
	//
	// If you do not update Duration, calling Span.End will calculate the
	// duration based on the elapsed time since the span's start time.
	Duration time.Duration

	// Outcome holds the span outcome: success, failure, or unknown (the default).
	// If Outcome is set to something else, it will be replaced with "unknown".
	//
	// Outcome is used for error rate calculations. A value of "success" indicates
	// that a operation succeeded, while "failure" indicates that the operation
	// failed. If Outcome is set to "unknown" (or some other value), then the
	// span will not be included in error rate calculations.
	Outcome string

	// Context describes the context in which span occurs.
	Context SpanContext

	stacktrace []stacktrace.Frame
}

func (s *SpanData) setStacktrace(skip int) {
	s.stacktrace = stacktrace.AppendStacktrace(s.stacktrace[:0], skip+1, s.stackTraceLimit)
}

func (s *SpanData) reset(tracer *Tracer) {
	*s = SpanData{
		Context:    s.Context,
		Duration:   -1,
		stacktrace: s.stacktrace[:0],
	}
	s.Context.reset()
	tracer.spanDataPool.Put(s)
}
