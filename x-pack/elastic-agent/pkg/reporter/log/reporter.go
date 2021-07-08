// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package log

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
)

type logger interface {
	Error(...interface{})
	Info(...interface{})
}

// Reporter is a reporter without any effects, serves just as a showcase for further implementations.
type Reporter struct {
	logger     logger
	formatFunc func(record reporter.Event) string
}

// NewReporter creates a new noop reporter
func NewReporter(l logger) *Reporter {
	return &Reporter{
		logger:     l,
		formatFunc: defaultFormatFunc,
	}
}

// Report in noop reporter does nothing
func (r *Reporter) Report(ctx context.Context, record reporter.Event) error {
	if record.Type() == reporter.EventTypeError {
		r.logger.Error(r.formatFunc(record))
		return nil
	}

	r.logger.Info(r.formatFunc(record))
	return nil
}

// Close stops all the background jobs reporter is running.
func (r *Reporter) Close() error { return nil }

func defaultFormatFunc(e reporter.Event) string {
	return fmt.Sprintf(defaultLogFormat,
		e.Time().Format(timeFormat),
		e.Message(),
		e.Type(),
		e.SubType(),
	)
}

// Check it is reporter.Backend
var _ reporter.Backend = &Reporter{}
