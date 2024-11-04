// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"

	"github.com/devigned/tab"

	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	tab.Register(new(logsOnlyTracer))
}

// logsOnlyTracer manages the creation of the required
// Spanners and Loggers with the goal of deferring logging
// to the `logp` package.
//
// According to the `github.com/devigned/tab package`,
// to implement a Tracer, you must provide the following
// three components:
//
// - Tracer
// - Spanner
// - Logger
//
// Since we are currently only interested in logging, we will
// implement a Tracer that only logs.
type logsOnlyTracer struct{}

// ----------------------------------------------------------------------------
// Tracer
// ----------------------------------------------------------------------------

// StartSpan returns the input context and a no-op Spanner
func (nt *logsOnlyTracer) StartSpan(ctx context.Context, operationName string, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(logsOnlySpanner)
}

// StartSpanWithRemoteParent returns the input context and a no-op Spanner
func (nt *logsOnlyTracer) StartSpanWithRemoteParent(ctx context.Context, operationName string, carrier tab.Carrier, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(logsOnlySpanner)
}

// FromContext returns a no-op Spanner without regard to the input context
func (nt *logsOnlyTracer) FromContext(ctx context.Context) tab.Spanner {
	return new(logsOnlySpanner)
}

// NewContext returns the parent context
func (nt *logsOnlyTracer) NewContext(parent context.Context, span tab.Spanner) context.Context {
	return parent
}

// ----------------------------------------------------------------------------
// Spanner
// ----------------------------------------------------------------------------

// logsOnlySpanner is a Spanner implementation that focuses
// on logging only.
type logsOnlySpanner struct{}

// AddAttributes is a no-op
func (ns *logsOnlySpanner) AddAttributes(attributes ...tab.Attribute) {}

// End is a no-op
func (ns *logsOnlySpanner) End() {}

// Logger returns a Logger implementation
func (ns *logsOnlySpanner) Logger() tab.Logger {
	return &logpLogger{logp.L()}
}

// Inject is no-op
func (ns *logsOnlySpanner) Inject(carrier tab.Carrier) error {
	return nil
}

// InternalSpan returns nil
func (ns *logsOnlySpanner) InternalSpan() interface{} {
	return nil
}

// ----------------------------------------------------------------------------
// Logger
// ----------------------------------------------------------------------------

// logpLogger defers logging to the logp package
type logpLogger struct {
	logger *logp.Logger
}

// Info logs a message at info level
func (sl logpLogger) Info(msg string, attributes ...tab.Attribute) {
	sl.logger.Info(msg)
}

// Error logs a message at error level
func (sl logpLogger) Error(err error, attributes ...tab.Attribute) {
	sl.logger.Error(err)
}

// Fatal logs a message at Fatal level
func (sl logpLogger) Fatal(msg string, attributes ...tab.Attribute) {
	sl.logger.Fatal(msg)
}

// Debug logs a message at Debug level
func (sl logpLogger) Debug(msg string, attributes ...tab.Attribute) {
	sl.logger.Debug(msg)
}
