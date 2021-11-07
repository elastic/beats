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
	"go.elastic.co/apm/internal/ringbuffer"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
	"go.elastic.co/fastjson"
)

const (
	transactionBlockTag ringbuffer.BlockTag = iota + 1
	spanBlockTag
	errorBlockTag
	metricsBlockTag
)

// notSampled is used as the pointee for the model.Transaction.Sampled field
// of non-sampled transactions.
var notSampled = false

type modelWriter struct {
	buffer          *ringbuffer.Buffer
	metricsBuffer   *ringbuffer.Buffer
	cfg             *tracerConfig
	stats           *TracerStats
	json            fastjson.Writer
	modelStacktrace []model.StacktraceFrame
}

// writeTransaction encodes tx as JSON to the buffer, and then resets tx.
func (w *modelWriter) writeTransaction(tx *Transaction, td *TransactionData) {
	var modelTx model.Transaction
	w.buildModelTransaction(&modelTx, tx, td)
	w.json.RawString(`{"transaction":`)
	modelTx.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), transactionBlockTag)
	w.json.Reset()
	td.reset(tx.tracer)
}

// writeSpan encodes s as JSON to the buffer, and then resets s.
func (w *modelWriter) writeSpan(s *Span, sd *SpanData) {
	var modelSpan model.Span
	w.buildModelSpan(&modelSpan, s, sd)
	w.json.RawString(`{"span":`)
	modelSpan.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), spanBlockTag)
	w.json.Reset()
	sd.reset(s.tracer)
}

// writeError encodes e as JSON to the buffer, and then resets e.
func (w *modelWriter) writeError(e *ErrorData) {
	var modelError model.Error
	w.buildModelError(&modelError, e)
	w.json.RawString(`{"error":`)
	modelError.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), errorBlockTag)
	w.json.Reset()
	e.reset()
}

// writeMetrics encodes m as JSON to the w.metricsBuffer, and then resets m.
//
// Note that we do not write metrics to the main ring buffer (w.buffer), as
// periodic metrics would be evicted by transactions/spans in a busy system.
func (w *modelWriter) writeMetrics(m *Metrics) {
	for _, m := range m.transactionGroupMetrics {
		w.json.RawString(`{"metricset":`)
		m.MarshalFastJSON(&w.json)
		w.json.RawString("}")
		w.metricsBuffer.WriteBlock(w.json.Bytes(), metricsBlockTag)
		w.json.Reset()
	}
	for _, m := range m.metrics {
		w.json.RawString(`{"metricset":`)
		m.MarshalFastJSON(&w.json)
		w.json.RawString("}")
		w.metricsBuffer.WriteBlock(w.json.Bytes(), metricsBlockTag)
		w.json.Reset()
	}
	m.reset()
}

func (w *modelWriter) buildModelTransaction(out *model.Transaction, tx *Transaction, td *TransactionData) {
	out.ID = model.SpanID(tx.traceContext.Span)
	out.TraceID = model.TraceID(tx.traceContext.Trace)
	sampled := tx.traceContext.Options.Recorded()
	if !sampled {
		out.Sampled = &notSampled
	}
	if tx.traceContext.State.haveSampleRate {
		out.SampleRate = &tx.traceContext.State.sampleRate
	}

	out.ParentID = model.SpanID(td.parentSpan)
	out.Name = truncateString(td.Name)
	out.Type = truncateString(td.Type)
	out.Result = truncateString(td.Result)
	out.Outcome = normalizeOutcome(td.Outcome)
	out.Timestamp = model.Time(td.timestamp.UTC())
	out.Duration = td.Duration.Seconds() * 1000
	out.SpanCount.Started = td.spansCreated
	out.SpanCount.Dropped = td.spansDropped
	if sampled {
		out.Context = td.Context.build()
	}
}

func (w *modelWriter) buildModelSpan(out *model.Span, span *Span, sd *SpanData) {
	w.modelStacktrace = w.modelStacktrace[:0]
	out.ID = model.SpanID(span.traceContext.Span)
	out.TraceID = model.TraceID(span.traceContext.Trace)
	out.TransactionID = model.SpanID(span.transactionID)
	if span.traceContext.State.haveSampleRate {
		out.SampleRate = &span.traceContext.State.sampleRate
	}

	out.ParentID = model.SpanID(sd.parentID)
	out.Name = truncateString(sd.Name)
	out.Type = truncateString(sd.Type)
	out.Subtype = truncateString(sd.Subtype)
	out.Action = truncateString(sd.Action)
	out.Timestamp = model.Time(sd.timestamp.UTC())
	out.Duration = sd.Duration.Seconds() * 1000
	out.Outcome = normalizeOutcome(sd.Outcome)
	out.Context = sd.Context.build()

	// Copy the span type to context.destination.service.type.
	if out.Context != nil && out.Context.Destination != nil && out.Context.Destination.Service != nil {
		out.Context.Destination.Service.Type = out.Type
	}

	w.modelStacktrace = appendModelStacktraceFrames(w.modelStacktrace, sd.stacktrace)
	out.Stacktrace = w.modelStacktrace
	w.setStacktraceContext(out.Stacktrace)
}

func (w *modelWriter) buildModelError(out *model.Error, e *ErrorData) {
	out.ID = model.TraceID(e.ID)
	out.TraceID = model.TraceID(e.TraceID)
	out.ParentID = model.SpanID(e.ParentID)
	out.TransactionID = model.SpanID(e.TransactionID)
	out.Timestamp = model.Time(e.Timestamp.UTC())
	out.Context = e.Context.build()
	out.Culprit = e.Culprit

	if !e.TransactionID.isZero() {
		out.Transaction.Sampled = &e.transactionSampled
		if e.transactionSampled {
			out.Transaction.Type = e.transactionType
		}
	}

	// Create model stacktrace frames, and set the context.
	w.modelStacktrace = w.modelStacktrace[:0]
	var appendModelErrorStacktraceFrames func(exception *exceptionData)
	appendModelErrorStacktraceFrames = func(exception *exceptionData) {
		if len(exception.stacktrace) != 0 {
			w.modelStacktrace = appendModelStacktraceFrames(w.modelStacktrace, exception.stacktrace)
		}
		for _, cause := range exception.cause {
			appendModelErrorStacktraceFrames(&cause)
		}
	}
	appendModelErrorStacktraceFrames(&e.exception)
	if len(e.logStacktrace) != 0 {
		w.modelStacktrace = appendModelStacktraceFrames(w.modelStacktrace, e.logStacktrace)
	}
	w.setStacktraceContext(w.modelStacktrace)

	var modelStacktraceOffset int
	if e.exception.message != "" {
		var buildException func(exception *exceptionData) model.Exception
		culprit := e.Culprit
		buildException = func(exception *exceptionData) model.Exception {
			out := model.Exception{
				Message: exception.message,
				Code: model.ExceptionCode{
					String: exception.Code.String,
					Number: exception.Code.Number,
				},
				Type:    exception.Type.Name,
				Module:  exception.Type.PackagePath,
				Handled: e.Handled,
			}
			if n := len(exception.stacktrace); n != 0 {
				out.Stacktrace = w.modelStacktrace[modelStacktraceOffset : modelStacktraceOffset+n]
				modelStacktraceOffset += n
			}
			if len(exception.attrs) != 0 {
				out.Attributes = exception.attrs
			}
			if n := len(exception.cause); n > 0 {
				out.Cause = make([]model.Exception, n)
				for i := range exception.cause {
					out.Cause[i] = buildException(&exception.cause[i])
				}
			}
			if culprit == "" {
				culprit = stacktraceCulprit(out.Stacktrace)
			}
			return out
		}
		out.Exception = buildException(&e.exception)
		out.Culprit = culprit
	}
	if e.log.Message != "" {
		out.Log = model.Log{
			Message:      e.log.Message,
			Level:        e.log.Level,
			LoggerName:   e.log.LoggerName,
			ParamMessage: e.log.MessageFormat,
		}
		if n := len(e.logStacktrace); n != 0 {
			out.Log.Stacktrace = w.modelStacktrace[modelStacktraceOffset : modelStacktraceOffset+n]
			modelStacktraceOffset += n
			if out.Culprit == "" {
				out.Culprit = stacktraceCulprit(out.Log.Stacktrace)
			}
		}
	}
	out.Culprit = truncateString(out.Culprit)
}

func stacktraceCulprit(frames []model.StacktraceFrame) string {
	for _, frame := range frames {
		if !frame.LibraryFrame {
			return frame.Function
		}
	}
	return ""
}

func (w *modelWriter) setStacktraceContext(stack []model.StacktraceFrame) {
	if w.cfg.contextSetter == nil || len(stack) == 0 {
		return
	}
	err := stacktrace.SetContext(w.cfg.contextSetter, stack, w.cfg.preContext, w.cfg.postContext)
	if err != nil {
		if w.cfg.logger != nil {
			w.cfg.logger.Debugf("setting context failed: %v", err)
		}
		w.stats.Errors.SetContext++
	}
}

func normalizeOutcome(outcome string) string {
	switch outcome {
	case "success", "failure", "unknown":
		return outcome
	default:
		return "unknown"
	}
}
