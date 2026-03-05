// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestTraceSpans_SingleExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{"hello":"world"}`)
	}))
	defer server.Close()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	defer tp.Shutdown(context.Background())

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"interval":     "1m",
		"resource.url": server.URL,
		"program": `
			bytes(get(state.url).Body).as(body, {
				"events": [body.decode_json()]
			})
		`,
	})
	c := defaultConfig()
	c.Redact = &redact{}
	if err := cfg.Unpack(&c); err != nil {
		t.Fatalf("unpacking config: %v", err)
	}

	// Cancel as soon as the first periodic run trace is complete.
	// The timeout is only a safety guard for regressions.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cancelWhenTraceEnds(ctx, cancel, sr, "cel.periodic.run")

	v2Ctx := v2.Context{
		Logger:          logp.NewLogger("cel_trace_test"),
		ID:              "trace_test:single",
		IDWithoutName:   "trace_test:single",
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}
	pub := &traceTestPublisher{done: func(int) {}}

	err := input{tracerProvider: tp}.run(v2Ctx, &source{c}, nil, pub, &v2Ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tp.ForceFlush(context.Background())
	spans := sr.Ended()

	runSpan := findSpan(spans, "cel.periodic.run")
	execSpan := findSpan(spans, "cel.program.execution")
	pubSpan := findSpan(spans, "cel.program.publish")
	if runSpan == nil {
		t.Fatal("missing cel.periodic.run span")
	}
	if execSpan == nil {
		t.Fatal("missing cel.program.execution span")
	}
	if pubSpan == nil {
		t.Fatal("missing cel.program.publish span")
	}

	if execSpan.Parent().TraceID() != runSpan.SpanContext().TraceID() {
		t.Error("execution span not in same trace as run span")
	}
	if execSpan.Parent().SpanID() != runSpan.SpanContext().SpanID() {
		t.Error("execution span is not a child of run span")
	}
	if pubSpan.Parent().SpanID() != execSpan.SpanContext().SpanID() {
		t.Error("publish span is not a child of execution span")
	}

	assertIntAttr(t, runSpan, "cel.periodic.execution_count", 1)
	assertIntAttr(t, runSpan, "cel.periodic.event_count", 1)
	assertBoolAttr(t, runSpan, "cel.periodic.max_execution_limited", false)
	assertIntAttr(t, execSpan, "cel.program.event_count", 1)
	assertBoolAttr(t, execSpan, "cel.program.want_more", false)
	assertIntAttr(t, execSpan, "cel.program.execution_number", 1)

	if runSpan.Status().Code != codes.Ok {
		t.Errorf("run span status = %v, want Ok", runSpan.Status().Code)
	}
	if execSpan.Status().Code != codes.Ok {
		t.Errorf("execution span status = %v, want Ok", execSpan.Status().Code)
	}
}

func TestTraceSpans_WantMore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{"item":"x"}`)
	}))
	defer server.Close()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	defer tp.Shutdown(context.Background())

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"interval":     "1m",
		"resource.url": server.URL,
		"program": `
			bytes(get(state.url).Body).as(body, {
				"events": [body.decode_json()],
				"want_more": int(state.?runcount.orValue(1)) < 2,
				"runcount": int(state.?runcount.orValue(1)) + 1,
			})
		`,
	})
	c := defaultConfig()
	c.Redact = &redact{}
	if err := cfg.Unpack(&c); err != nil {
		t.Fatalf("unpacking config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Cancel as soon as the periodic run trace is complete.
	// The timeout is only a safety guard for regressions.
	cancelWhenTraceEnds(ctx, cancel, sr, "cel.periodic.run")

	v2Ctx := v2.Context{
		Logger:          logp.NewLogger("cel_trace_test"),
		ID:              "trace_test:want_more",
		IDWithoutName:   "trace_test:want_more",
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}
	pub := &traceTestPublisher{done: func(int) {}}

	err := input{tracerProvider: tp}.run(v2Ctx, &source{c}, nil, pub, &v2Ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tp.ForceFlush(context.Background())
	spans := sr.Ended()

	runSpans := findSpans(spans, "cel.periodic.run")
	execSpans := findSpans(spans, "cel.program.execution")

	if len(runSpans) != 1 {
		t.Fatalf("got %d run spans, want 1", len(runSpans))
	}
	if len(execSpans) != 2 {
		t.Fatalf("got %d execution spans, want 2", len(execSpans))
	}

	run := runSpans[0]
	for i, exec := range execSpans {
		if exec.Parent().SpanID() != run.SpanContext().SpanID() {
			t.Errorf("execution span %d is not a child of run span", i)
		}
	}

	assertIntAttr(t, run, "cel.periodic.execution_count", 2)
	assertIntAttr(t, run, "cel.periodic.event_count", 2)
	assertBoolAttr(t, execSpans[0], "cel.program.want_more", true)
	assertBoolAttr(t, execSpans[1], "cel.program.want_more", false)
	assertIntAttr(t, execSpans[0], "cel.program.execution_number", 1)
	assertIntAttr(t, execSpans[1], "cel.program.execution_number", 2)
}

func TestTraceSpans_EvalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	defer tp.Shutdown(context.Background())

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"interval":     1,
		"resource.url": server.URL,
		"program": `
			bytes(get(state.url).Body).as(body, {
				"events": [body.decode_json()]
			})
		`,
	})
	c := defaultConfig()
	c.Redact = &redact{}
	if err := cfg.Unpack(&c); err != nil {
		t.Fatalf("unpacking config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v2Ctx := v2.Context{
		Logger:          logp.NewLogger("cel_trace_test"),
		ID:              "trace_test:error",
		IDWithoutName:   "trace_test:error",
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}
	pub := &traceTestPublisher{done: func(int) { cancel() }}

	_ = input{tracerProvider: tp}.run(v2Ctx, &source{c}, nil, pub, &v2Ctx)

	tp.ForceFlush(context.Background())
	spans := sr.Ended()

	runSpan := findSpan(spans, "cel.periodic.run")
	execSpan := findSpan(spans, "cel.program.execution")
	if runSpan == nil {
		t.Fatal("missing cel.periodic.run span")
	}
	if execSpan == nil {
		t.Fatal("missing cel.program.execution span")
	}

	if execSpan.Status().Code != codes.Error {
		t.Errorf("execution span status = %v, want Error", execSpan.Status().Code)
	}
	if runSpan.Status().Code != codes.Error {
		t.Errorf("run span status = %v, want Error", runSpan.Status().Code)
	}

	// All spans must be ended (the SpanRecorder only records ended spans,
	// so their presence here is the assertion).
	for _, s := range spans {
		if s.EndTime().IsZero() {
			t.Errorf("span %q has zero end time", s.Name())
		}
	}
}

// traceTestPublisher is a minimal publisher for trace tests.
type traceTestPublisher struct {
	mu   sync.Mutex
	n    int
	done func(n int)
}

func (p *traceTestPublisher) Publish(_ beat.Event, _ interface{}) error {
	p.mu.Lock()
	p.n++
	n := p.n
	p.mu.Unlock()
	p.done(n)
	return nil
}

func cancelWhenTraceEnds(ctx context.Context, cancel context.CancelFunc, sr *tracetest.SpanRecorder, rootSpanName string) {
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, s := range sr.Ended() {
					if s.Name() == rootSpanName {
						cancel()
						return
					}
				}
			}
		}
	}()
}

func findSpan(spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	for _, s := range spans {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func findSpans(spans []sdktrace.ReadOnlySpan, name string) []sdktrace.ReadOnlySpan {
	var out []sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == name {
			out = append(out, s)
		}
	}
	return out
}

func assertIntAttr(t *testing.T, s sdktrace.ReadOnlySpan, key string, want int) {
	t.Helper()
	for _, a := range s.Attributes() {
		if string(a.Key) == key {
			if got := a.Value.AsInt64(); got != int64(want) {
				t.Errorf("span %q attr %q = %d, want %d", s.Name(), key, got, want)
			}
			return
		}
	}
	t.Errorf("span %q missing attribute %q", s.Name(), key)
}

func assertBoolAttr(t *testing.T, s sdktrace.ReadOnlySpan, key string, want bool) {
	t.Helper()
	for _, a := range s.Attributes() {
		if string(a.Key) == key {
			if got := a.Value.AsBool(); got != want {
				t.Errorf("span %q attr %q = %v, want %v", s.Name(), key, got, want)
			}
			return
		}
	}
	t.Errorf("span %q missing attribute %q", s.Name(), key)
}
