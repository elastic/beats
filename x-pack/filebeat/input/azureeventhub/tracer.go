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
	tab.Register(new(tracer))
}

// tracer manages the creation of spanners
type tracer struct{}

// StartSpan returns the input context and a no-op Spanner
func (nt *tracer) StartSpan(ctx context.Context, operationName string, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(logOnlySpanner)
}

// StartSpanWithRemoteParent returns the input context and a no-op Spanner
func (nt *tracer) StartSpanWithRemoteParent(ctx context.Context, operationName string, carrier tab.Carrier, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(logOnlySpanner)
}

// FromContext returns a no-op Spanner without regard to the input context
func (nt *tracer) FromContext(ctx context.Context) tab.Spanner {
	return new(logOnlySpanner)
}

// NewContext returns the parent context
func (nt *tracer) NewContext(parent context.Context, span tab.Spanner) context.Context {
	return parent
}

// logOnlySpanner is a Spanner implementation that focuses
// on logging only.
type logOnlySpanner struct{}

// AddAttributes is a no-op
func (ns *logOnlySpanner) AddAttributes(attributes ...tab.Attribute) {}

// End is a no-op
func (ns *logOnlySpanner) End() {}

// Logger returns a Logger implementation
func (ns *logOnlySpanner) Logger() tab.Logger {
	return &logpLogger{logp.L()}
}

// Inject is no-op
func (ns *logOnlySpanner) Inject(carrier tab.Carrier) error {
	return nil
}

// InternalSpan returns nil
func (ns *logOnlySpanner) InternalSpan() interface{} {
	return nil
}

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
