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
	tab.Register(&tracer{})
}

type tracer struct{}

type noOpSpanner struct{}

type logpLogger struct{}

// StartSpan returns the input context and a no op Spanner
func (nt *tracer) StartSpan(ctx context.Context, operationName string, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(noOpSpanner)
}

// StartSpanWithRemoteParent returns the input context and a no op Spanner
func (nt *tracer) StartSpanWithRemoteParent(ctx context.Context, operationName string, carrier tab.Carrier, opts ...interface{}) (context.Context, tab.Spanner) {
	return ctx, new(noOpSpanner)
}

// FromContext returns a no op Spanner without regard to the input context
func (nt *tracer) FromContext(ctx context.Context) tab.Spanner {
	return new(noOpSpanner)
}

// NewContext returns the parent context
func (nt *tracer) NewContext(parent context.Context, span tab.Spanner) context.Context {
	return parent
}

// AddAttributes is a nop
func (ns *noOpSpanner) AddAttributes(attributes ...tab.Attribute) {}

// End is a nop
func (ns *noOpSpanner) End() {}

// Logger returns a nopLogger
func (ns *noOpSpanner) Logger() tab.Logger {
	return new(logpLogger)
}

// Inject is a nop
func (ns *noOpSpanner) Inject(carrier tab.Carrier) error {
	return nil
}

// InternalSpan returns nil
func (ns *noOpSpanner) InternalSpan() interface{} {
	return nil
}

// Info nops log entry
func (sl logpLogger) Info(msg string, attributes ...tab.Attribute) {
	logp.L().Info(msg)
}

// Error nops log entry
func (sl logpLogger) Error(err error, attributes ...tab.Attribute) {
	logp.L().Error(err)
}

// Fatal nops log entry
func (sl logpLogger) Fatal(msg string, attributes ...tab.Attribute) {
	logp.L().Fatal(msg)
}

// Debug nops log entry
func (sl logpLogger) Debug(msg string, attributes ...tab.Attribute) {
	logp.L().Debug(msg)
}
