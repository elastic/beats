// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
// This file was contributed to by generative AI

package fbreceiver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

const overlapWarningText = "harvested by more than one receiver"

type overlapEnv struct {
	t         *testing.T
	dataDir   string
	logDir    string
	factory   receiver.Factory
	mu        sync.Mutex
	receivers []*overlapReceiver
}

type overlapReceiver struct {
	env     *overlapEnv
	name    string
	rcvr    receiver.Logs
	host    *oteltest.MockHost
	logs    *observer.ObservedLogs
	mu      sync.Mutex
	lines   []string
	stopped bool
}

func newOverlapEnv(t *testing.T) *overlapEnv {
	return &overlapEnv{
		t:       t,
		dataDir: t.TempDir(),
		logDir:  t.TempDir(),
		factory: NewFactoryWithSettings(Settings{Home: t.TempDir()}),
	}
}

func (e *overlapEnv) path(name string) string { return filepath.Join(e.logDir, name) }

func (e *overlapEnv) append(name string, lines ...string) {
	for _, l := range lines {
		writeFile(e.t, e.path(name), l)
	}
}

func overlapLines(prefix string, from, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s-%03d", prefix, from+i)
	}
	return out
}

func (e *overlapEnv) start(name string, inputPaths ...[]string) *overlapReceiver {
	t := e.t
	t.Helper()
	inputs := make([]map[string]any, len(inputPaths))
	for i, p := range inputPaths {
		inputs[i] = map[string]any{
			"type":                 "log",
			"allow_deprecated_use": true,
			"paths":                p,
			"scan_frequency":       "100ms",
			"fields":               map[string]any{"input": i},
		}
	}
	cfg := &Config{Beatconfig: map[string]any{
		"queue.mem.flush.timeout": "0s",
		"filebeat": map[string]any{
			"registry": map[string]any{"flush": "10ms"},
			"inputs":   inputs,
		},
		"logging":                 map[string]any{"level": "debug"},
		"path.home":               t.TempDir(),
		"path.data":               e.dataDir,
		"http.enabled":            false,
		"management.otel.enabled": true,
	}}

	r := &overlapReceiver{env: e, name: name, host: &oteltest.MockHost{}}
	var core zapcore.Core
	core, r.logs = observer.New(zapcore.DebugLevel)
	var settings receiver.Settings
	settings.ID = component.NewIDWithName(e.factory.Type(), name)
	settings.Logger = zap.New(core.With([]zapcore.Field{
		zap.String("otelcol.component.id", settings.ID.String()),
		zap.String("otelcol.component.kind", "receiver"),
		zap.String("otelcol.signal", "logs"),
	}))
	sink, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
		for _, rl := range ld.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					msg, _ := lr.Body().Map().AsRaw()["message"].(string)
					r.mu.Lock()
					r.lines = append(r.lines, msg)
					r.mu.Unlock()
				}
			}
		}
		return nil
	})
	require.NoError(t, err, "consumer for %s must be created", name)
	r.rcvr, err = e.factory.CreateLogs(t.Context(), settings, cfg, sink)
	require.NoError(t, err, "receiver %s must be created", name)
	require.NoError(t, r.rcvr.Start(t.Context(), r.host), "receiver %s must start", name)
	e.mu.Lock()
	e.receivers = append(e.receivers, r)
	e.mu.Unlock()
	t.Cleanup(r.stop)
	return r
}

func (r *overlapReceiver) stop() {
	if r.stopped {
		return
	}
	r.stopped = true
	require.NoError(r.env.t, r.rcvr.Shutdown(r.env.t.Context()), "receiver %s must stop", r.name)
}

func (r *overlapReceiver) ingested() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

func (r *overlapReceiver) waitIngested(n int) {
	r.env.t.Helper()
	ok := assert.Eventually(r.env.t, func() bool { return len(r.ingested()) >= n }, 30*time.Second, 20*time.Millisecond)
	if !ok {
		r.dumpLogs()
	}
	require.True(r.env.t, ok, "receiver %s must ingest %d records, has %d", r.name, n, len(r.ingested()))
}

func (r *overlapReceiver) dumpLogs() {
	t := r.env.t
	t.Helper()
	for _, rc := range r.env.receivers {
		for _, e := range rc.logs.All() {
			t.Logf("[%s] %s %s %v", rc.name, e.Time.Format("15:04:05.000"), e.Message, e.ContextMap())
		}
	}
	entries, _ := os.ReadDir(r.env.logDir)
	for _, de := range entries {
		if info, err := de.Info(); err == nil {
			t.Logf("log file %s size=%d", de.Name(), info.Size())
		}
	}
	t.Logf("registry offsets: %v", registryOffsets(t, r.env.dataDir))
}

func (e *overlapEnv) flushes() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for _, r := range e.receivers {
		n += r.logs.FilterMessageSnippet("Registry file updated").Len()
	}
	return n
}

func (r *overlapReceiver) ingestAndFlush(n int) {
	t := r.env.t
	t.Helper()
	r.waitIngested(n)
	last, stable := -1, 0
	require.Eventually(t, func() bool {
		cur := r.env.flushes()
		if cur == last {
			stable++
		} else {
			last, stable = cur, 0
		}
		return stable >= 5
	}, 30*time.Second, 20*time.Millisecond, "registry flushes must settle")
}

func (r *overlapReceiver) warnings() int {
	return r.logs.FilterMessageSnippet(overlapWarningText).Len()
}

func (r *overlapReceiver) assertHealthy() {
	r.env.t.Helper()
	for _, e := range r.host.GetEvents() {
		assert.NotEqual(r.env.t, componentstatus.StatusPermanentError, e.Status(), "receiver %s must not report a permanent error: %v", r.name, e.Err())
		assert.NotEqual(r.env.t, componentstatus.StatusFatalError, e.Status(), "receiver %s must not report a fatal error: %v", r.name, e.Err())
	}
}

func (e *overlapEnv) assertOffsetsAtEnd(names ...string) {
	e.t.Helper()
	offsets := registryOffsets(e.t, e.dataDir)
	for _, n := range names {
		info, err := os.Stat(e.path(n))
		require.NoError(e.t, err, "log file must exist")
		assert.Equal(e.t, info.Size(), offsets[e.path(n)], "registry must resume %s at its end", n)
	}
}

func assertOverlapWarnings(t *testing.T, want int, rs ...*overlapReceiver) {
	t.Helper()
	if os.Getenv("LOG_OVERLAP_BASELINE") == "1" {
		return
	}
	for _, r := range rs {
		assert.Equal(t, want, r.warnings(), "receiver %s overlap warnings", r.name)
	}
}

func TestLogOverlapN1DisjointFilesRestart(t *testing.T) {
	e := newOverlapEnv(t)
	a, b := e.path("a.log"), e.path("b.log")
	e.append("a.log", overlapLines("a", 0, 5)...)
	e.append("b.log", overlapLines("b", 0, 5)...)
	ra, rb := e.start("a", []string{a}), e.start("b", []string{b})
	ra.ingestAndFlush(5)
	rb.ingestAndFlush(5)
	ra.stop()
	rb.stop()

	e.append("a.log", overlapLines("a", 5, 3)...)
	e.append("b.log", overlapLines("b", 5, 3)...)
	ra2, rb2 := e.start("a2", []string{a}), e.start("b2", []string{b})
	ra2.ingestAndFlush(3)
	rb2.ingestAndFlush(3)
	assert.ElementsMatch(t, overlapLines("a", 5, 3), ra2.ingested())
	assert.ElementsMatch(t, overlapLines("b", 5, 3), rb2.ingested())
	ra2.assertHealthy()
	rb2.assertHealthy()
	assertOverlapWarnings(t, 0, ra, rb, ra2, rb2)
	ra2.stop()
	rb2.stop()
	e.assertOffsetsAtEnd("a.log", "b.log")
}

func TestLogOverlapN2DisjointReceiverAddedWhileRunning(t *testing.T) {
	e := newOverlapEnv(t)
	e.append("a.log", overlapLines("a", 0, 5)...)
	e.append("b.log", overlapLines("b", 0, 4)...)
	ra := e.start("a", []string{e.path("a.log")})
	ra.ingestAndFlush(5)
	rb := e.start("b", []string{e.path("b.log")})
	rb.ingestAndFlush(4)
	assert.ElementsMatch(t, overlapLines("a", 0, 5), ra.ingested())
	assert.ElementsMatch(t, overlapLines("b", 0, 4), rb.ingested())
	ra.assertHealthy()
	rb.assertHealthy()
	assertOverlapWarnings(t, 0, ra, rb)
	ra.stop()
	rb.stop()
	e.assertOffsetsAtEnd("a.log", "b.log")
}

func TestLogOverlapO1BothStartTogetherFileCreatedAfter(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra, rb := e.start("a", p), e.start("b", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	rb.ingestAndFlush(5)
	assert.ElementsMatch(t, overlapLines("x", 0, 5), ra.ingested())
	assert.ElementsMatch(t, overlapLines("x", 0, 5), rb.ingested())
	ra.assertHealthy()
	rb.assertHealthy()
	assertOverlapWarnings(t, 1, ra, rb)
}

func TestLogOverlapO2RestartWithExistingRegistry(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra, rb := e.start("a", p), e.start("b", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	rb.ingestAndFlush(5)
	ra.stop()
	rb.stop()

	e.append("x.log", overlapLines("x", 5, 3)...)
	ra2, rb2 := e.start("a2", p), e.start("b2", p)
	ra2.ingestAndFlush(3)
	rb2.ingestAndFlush(3)
	assert.ElementsMatch(t, overlapLines("x", 5, 3), ra2.ingested())
	assert.ElementsMatch(t, overlapLines("x", 5, 3), rb2.ingested())
	ra2.assertHealthy()
	rb2.assertHealthy()
	assertOverlapWarnings(t, 1, ra2, rb2)
}

func TestLogOverlapO3ReceiverAddedWhileOtherHoldsFile(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra := e.start("a", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	rb := e.start("b", p)

	e.append("x.log", overlapLines("x", 5, 3)...)
	ra.ingestAndFlush(8)
	rb.ingestAndFlush(3)
	assert.ElementsMatch(t, overlapLines("x", 0, 8), ra.ingested())
	assert.ElementsMatch(t, overlapLines("x", 5, 3), rb.ingested())
	ra.assertHealthy()
	rb.assertHealthy()
	assertOverlapWarnings(t, 1, ra, rb)
}

func TestLogOverlapO4WildcardReceiverAddedOverlapsOneFile(t *testing.T) {
	e := newOverlapEnv(t)
	ra := e.start("a", []string{e.path("a.log")})
	e.append("a.log", overlapLines("a", 0, 5)...)
	ra.ingestAndFlush(5)
	e.append("c.log", overlapLines("c", 0, 4)...)
	rb := e.start("b", []string{filepath.Join(e.logDir, "*.log")})
	rb.ingestAndFlush(4)

	e.append("a.log", overlapLines("a", 5, 3)...)
	ra.ingestAndFlush(8)
	rb.ingestAndFlush(7)
	assert.ElementsMatch(t, append(overlapLines("c", 0, 4), overlapLines("a", 5, 3)...), rb.ingested())
	ra.assertHealthy()
	rb.assertHealthy()
	assertOverlapWarnings(t, 1, ra, rb)
}

func TestLogOverlapO5OverlappingReceiverRestartedWhileOtherRuns(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra := e.start("a", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	rb := e.start("b", p)
	e.append("x.log", overlapLines("x", 5, 3)...)
	ra.ingestAndFlush(8)
	rb.ingestAndFlush(3)
	rb.stop()

	e.append("x.log", overlapLines("x", 8, 2)...)
	ra.ingestAndFlush(10)
	rb2 := e.start("b2", p)
	e.append("x.log", overlapLines("x", 10, 2)...)
	ra.ingestAndFlush(12)
	rb2.waitIngested(2)
	assert.ElementsMatch(t, overlapLines("x", 10, 2), rb2.ingested()[len(rb2.ingested())-2:])
	assert.Len(t, rb2.ingested(), 2)
	ra.assertHealthy()
	rb2.assertHealthy()
	assertOverlapWarnings(t, 2, ra)
	assertOverlapWarnings(t, 1, rb2)
}

func TestLogOverlapO6SequentialHandoffDoesNotWarn(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra := e.start("a", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	ra.stop()

	rb := e.start("b", p)
	e.append("x.log", overlapLines("x", 5, 3)...)
	rb.ingestAndFlush(3)
	assert.ElementsMatch(t, overlapLines("x", 5, 3), rb.ingested())
	ra.assertHealthy()
	rb.assertHealthy()
	assertOverlapWarnings(t, 0, ra, rb)
}

func TestLogOverlapS1OneReceiverTwoInputsSameFile(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra := e.start("a", p, p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(10)
	assert.Len(t, ra.ingested(), 10)
	ra.assertHealthy()
	assertOverlapWarnings(t, 0, ra)
}

func TestLogOverlapS2OneReceiverRestart(t *testing.T) {
	e := newOverlapEnv(t)
	p := []string{e.path("x.log")}
	ra := e.start("a", p)
	e.append("x.log", overlapLines("x", 0, 5)...)
	ra.ingestAndFlush(5)
	ra.stop()

	e.append("x.log", overlapLines("x", 5, 3)...)
	ra2 := e.start("a2", p)
	ra2.ingestAndFlush(3)
	assert.ElementsMatch(t, overlapLines("x", 5, 3), ra2.ingested())
	ra2.assertHealthy()
	assertOverlapWarnings(t, 0, ra, ra2)
	ra2.stop()
	e.assertOffsetsAtEnd("x.log")
}
