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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

type Fetcher interface {
	Module() mb.Module
	Name() string

	Fetch() ([]mb.Event, []error)
	WriteEvents(testing.TB, string)
	WriteEventsCond(testing.TB, string, func(common.MapStr) bool)
}

func NewFetcher(t testing.TB, config interface{}) Fetcher {
	metricSet := NewMetricSet(t, config)
	switch metricSet := metricSet.(type) {
	case mb.ReportingMetricSetV2:
		return NewReporterV2Fetcher(metricSet)
	case mb.ReportingMetricSetV2Error:
		return NewReporterV2FetcherError(metricSet)
	case mb.ReportingMetricSetV2WithContext:
		return NewReporterV2FetcherWithContext(metricSet)
	default:
		t.Fatalf("Failed to create a Fetcher for metricset of type %T", metricSet)
	}
	return nil
}

type reportingMetricSetV2Fetcher struct {
	fetcherHelper
	metricSet mb.ReportingMetricSetV2
}

func NewReporterV2Fetcher(metricSet mb.ReportingMetricSetV2) *reportingMetricSetV2Fetcher {
	return &reportingMetricSetV2Fetcher{
		fetcherHelper{metricSet},
		metricSet,
	}
}

func (f *reportingMetricSetV2Fetcher) Fetch() ([]mb.Event, []error) {
	return ReportingFetchV2(f.metricSet)
}

func (f *reportingMetricSetV2Fetcher) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2Fetcher) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	err := WriteEventsReporterV2Cond(f.metricSet, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

type reportingMetricSetV2FetcherError struct {
	fetcherHelper
	metricSet mb.ReportingMetricSetV2Error
}

func NewReporterV2FetcherError(metricSet mb.ReportingMetricSetV2Error) *reportingMetricSetV2FetcherError {
	return &reportingMetricSetV2FetcherError{
		fetcherHelper{metricSet},
		metricSet,
	}
}

func (f *reportingMetricSetV2FetcherError) Fetch() ([]mb.Event, []error) {
	return ReportingFetchV2Error(f.metricSet)
}

func (f *reportingMetricSetV2FetcherError) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2FetcherError) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	err := WriteEventsReporterV2ErrorCond(f.metricSet, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

type reportingMetricSetV2FetcherWithContext struct {
	fetcherHelper
	metricSet mb.ReportingMetricSetV2WithContext
}

func NewReporterV2FetcherWithContext(metricSet mb.ReportingMetricSetV2WithContext) *reportingMetricSetV2FetcherWithContext {
	return &reportingMetricSetV2FetcherWithContext{
		fetcherHelper{metricSet},
		metricSet,
	}
}

func (f *reportingMetricSetV2FetcherWithContext) Fetch() ([]mb.Event, []error) {
	return ReportingFetchV2WithContext(f.metricSet)
}

func (f *reportingMetricSetV2FetcherWithContext) WriteEvents(t testing.TB, path string) {
	f.WriteEventsCond(t, path, nil)
}

func (f *reportingMetricSetV2FetcherWithContext) WriteEventsCond(t testing.TB, path string, cond func(common.MapStr) bool) {
	err := WriteEventsReporterV2WithContextCond(f.metricSet, t, path, cond)
	if err != nil {
		t.Fatal("writing events", err)
	}
}

type fetcherHelper struct {
	metricSet mb.MetricSet
}

func (f *fetcherHelper) Module() mb.Module {
	return f.metricSet.Module()
}

func (f *fetcherHelper) Name() string {
	return f.metricSet.Name()
}
