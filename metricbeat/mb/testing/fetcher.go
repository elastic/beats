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

package testing

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// Fetcher is an interface implemented by all fetchers for testing purpouses
type Fetcher interface {
	Module() mb.Module
	Name() string

	FetchEvents() ([]mb.Event, []error)
	WriteEvents(testing.TB, string)
	WriteEventsCond(testing.TB, string, func(common.MapStr) bool)
	StandardizeEvent(mb.Event, ...mb.EventModifier) beat.Event
}

// NewFetcher returns a test fetcher from a Metricset configuration
func NewFetcher(t testing.TB, config interface{}) Fetcher {
	metricSet := NewMetricSet(t, config)
	switch metricSet := metricSet.(type) {
	case mb.ReportingMetricSetV2:
		return newReporterV2Fetcher(metricSet)
	case mb.ReportingMetricSetV2Error:
		return newReporterV2FetcherError(metricSet)
	case mb.ReportingMetricSetV2WithContext:
		return newReporterV2FetcherWithContext(metricSet)
	default:
		t.Fatalf("Failed to create a Fetcher for metricset of type %T", metricSet)
	}
	return nil
}

type reportingMetricSetV2Fetcher struct {
	mb.ReportingMetricSetV2
}

func newReporterV2Fetcher(metricSet mb.ReportingMetricSetV2) *reportingMetricSetV2Fetcher {
	return &reportingMetricSetV2Fetcher{metricSet}
}

func (f *reportingMetricSetV2Fetcher) FetchEvents() ([]mb.Event, []error) {
	return ReportingFetchV2(f)
}

func (f *reportingMetricSetV2Fetcher) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2Fetcher) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	err := WriteEventsReporterV2Cond(f, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

func (f *reportingMetricSetV2Fetcher) StandardizeEvent(event mb.Event, modifiers ...mb.EventModifier) beat.Event {
	return StandardizeEvent(f, event, modifiers...)
}

type reportingMetricSetV2FetcherError struct {
	mb.ReportingMetricSetV2Error
}

func newReporterV2FetcherError(metricSet mb.ReportingMetricSetV2Error) *reportingMetricSetV2FetcherError {
	return &reportingMetricSetV2FetcherError{metricSet}
}

func (f *reportingMetricSetV2FetcherError) FetchEvents() ([]mb.Event, []error) {
	return ReportingFetchV2Error(f)
}

func (f *reportingMetricSetV2FetcherError) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2FetcherError) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	t.Helper()

	err := WriteEventsReporterV2ErrorCond(f, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

func (f *reportingMetricSetV2FetcherError) StandardizeEvent(event mb.Event, modifiers ...mb.EventModifier) beat.Event {
	return StandardizeEvent(f, event, modifiers...)
}

type reportingMetricSetV2FetcherWithContext struct {
	mb.ReportingMetricSetV2WithContext
}

func newReporterV2FetcherWithContext(metricSet mb.ReportingMetricSetV2WithContext) *reportingMetricSetV2FetcherWithContext {
	return &reportingMetricSetV2FetcherWithContext{metricSet}
}

func (f *reportingMetricSetV2FetcherWithContext) FetchEvents() ([]mb.Event, []error) {
	return ReportingFetchV2WithContext(f)
}

func (f *reportingMetricSetV2FetcherWithContext) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2FetcherWithContext) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	err := WriteEventsReporterV2WithContextCond(f, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

func (f *reportingMetricSetV2FetcherWithContext) StandardizeEvent(event mb.Event, modifiers ...mb.EventModifier) beat.Event {
	return StandardizeEvent(f, event, modifiers...)
}
