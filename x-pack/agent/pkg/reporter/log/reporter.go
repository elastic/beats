// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package log

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/reporter"
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
func NewReporter(l logger, cfg *Config) *Reporter {
	format := DefaultFormat
	if cfg != nil {
		format = cfg.Format
	}

	formatFunc := defaultFormatFunc
	if format == JSONFormat {
		formatFunc = jsonFormatFunc
	}

	return &Reporter{
		logger:     l,
		formatFunc: formatFunc,
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
		e.Type(),
		e.SubType(),
		e.Message(),
	)
}

func jsonFormatFunc(record reporter.Event) string {
	b, _ := json.Marshal(makeEventReportable(record))
	return string(b)
}

type reportableEvent struct {
	Type    string
	SubType string
	Time    string
	Message string
	Payload map[string]interface{} `json:"payload,omitempty"`
}

func makeEventReportable(event reporter.Event) reportableEvent {
	return reportableEvent{
		Type:    event.Type(),
		SubType: event.SubType(),
		Time:    event.Time().Format(timeFormat),
		Message: event.Message(),
		Payload: event.Payload(),
	}
}

// Check it is reporter.Backend
var _ reporter.Backend = &Reporter{}
