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

/*
Package testing provides utility functions for testing Module and MetricSet
implementations.

MetricSet Example

This is an example showing how to use this package to test a MetricSet. By
using these methods you ensure the MetricSet is instantiated in the same way
that Metricbeat does it and with the same validations.

	package mymetricset_test

	import (
		mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	)

	func TestFetch(t *testing.T) {
		f := mbtest.NewEventFetcher(t, getConfig())
		event, err := f.Fetch()
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

		// Test event attributes...
	}

	func getConfig() map[string]interface{} {
		return map[string]interface{}{
			"module":     "mymodule",
			"metricsets": []string{"status"},
			"hosts":      []string{mymodule.GetHostFromEnv()},
		}
	}
*/
package testing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

type TestModule struct {
	ModName   string
	ModConfig mb.ModuleConfig
	RawConfig *common.Config
}

func (m *TestModule) Name() string                      { return m.ModName }
func (m *TestModule) Config() mb.ModuleConfig           { return m.ModConfig }
func (m *TestModule) UnpackConfig(to interface{}) error { return m.RawConfig.Unpack(to) }

func NewTestModule(t testing.TB, config interface{}) *TestModule {
	c, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatal(err)
	}

	return &TestModule{RawConfig: c}
}

// NewMetricSet instantiates a new MetricSet using the given configuration.
// The ModuleFactory and MetricSetFactory are obtained from the global
// Registry.
func NewMetricSet(t testing.TB, config interface{}) mb.MetricSet {
	c, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatal(err)
	}
	m, metricsets, err := mb.NewModule(c, mb.Registry)
	if err != nil {
		t.Fatal("failed to create new MetricSet", err)
	}
	if m == nil {
		t.Fatal("no module instantiated")
	}

	if len(metricsets) != 1 {
		t.Fatal("invalid number of metricsets instantiated")
	}

	metricset := metricsets[0]
	if metricset == nil {
		t.Fatal("metricset is nil")
	}
	return metricset
}

// NewEventFetcher instantiates a new EventFetcher using the given
// configuration. The ModuleFactory and MetricSetFactory are obtained from the
// global Registry.
func NewEventFetcher(t testing.TB, config interface{}) mb.EventFetcher {
	metricSet := NewMetricSet(t, config)

	fetcher, ok := metricSet.(mb.EventFetcher)
	if !ok {
		t.Fatal("MetricSet does not implement EventFetcher")
	}

	return fetcher
}

// NewEventsFetcher instantiates a new EventsFetcher using the given
// configuration. The ModuleFactory and MetricSetFactory are obtained from the
// global Registry.
func NewEventsFetcher(t testing.TB, config interface{}) mb.EventsFetcher {
	metricSet := NewMetricSet(t, config)

	fetcher, ok := metricSet.(mb.EventsFetcher)
	if !ok {
		t.Fatal("MetricSet does not implement EventsFetcher")
	}

	return fetcher
}

func NewReportingMetricSet(t testing.TB, config interface{}) mb.ReportingMetricSet {
	metricSet := NewMetricSet(t, config)

	reportingMetricSet, ok := metricSet.(mb.ReportingMetricSet)
	if !ok {
		t.Fatal("MetricSet does not implement ReportingMetricSet")
	}

	return reportingMetricSet
}

// ReportingFetch runs the given reporting metricset and returns all of the
// events and errors that occur during that period.
func ReportingFetch(metricSet mb.ReportingMetricSet) ([]common.MapStr, []error) {
	r := &capturingReporter{}
	metricSet.Fetch(r)
	return r.events, r.errs
}

// NewReportingMetricSetV2 returns a new ReportingMetricSetV2 instance. Then
// you can use ReportingFetchV2 to perform a Fetch operation with the MetricSet.
func NewReportingMetricSetV2(t testing.TB, config interface{}) mb.ReportingMetricSetV2 {
	metricSet := NewMetricSet(t, config)

	reportingMetricSetV2, ok := metricSet.(mb.ReportingMetricSetV2)
	if !ok {
		t.Fatal("MetricSet does not implement ReportingMetricSetV2")
	}

	return reportingMetricSetV2
}

// NewReportingMetricSetV2Error returns a new ReportingMetricSetV2 instance. Then
// you can use ReportingFetchV2 to perform a Fetch operation with the MetricSet.
func NewReportingMetricSetV2Error(t testing.TB, config interface{}) mb.ReportingMetricSetV2Error {
	metricSet := NewMetricSet(t, config)

	reportingMetricSetV2Error, ok := metricSet.(mb.ReportingMetricSetV2Error)
	if !ok {
		t.Fatal("MetricSet does not implement ReportingMetricSetV2Error")
	}

	return reportingMetricSetV2Error
}

// NewReportingMetricSetV2WithContext returns a new ReportingMetricSetV2WithContext instance. Then
// you can use ReportingFetchV2 to perform a Fetch operation with the MetricSet.
func NewReportingMetricSetV2WithContext(t testing.TB, config interface{}) mb.ReportingMetricSetV2WithContext {
	metricSet := NewMetricSet(t, config)

	reportingMetricSet, ok := metricSet.(mb.ReportingMetricSetV2WithContext)
	if !ok {
		t.Fatal("MetricSet does not implement ReportingMetricSetV2WithContext")
	}

	return reportingMetricSet
}

// CapturingReporterV2 is a reporter used for testing which stores all events and errors
type CapturingReporterV2 struct {
	events []mb.Event
	errs   []error
}

// Event is used to report an event
func (r *CapturingReporterV2) Event(event mb.Event) bool {
	r.events = append(r.events, event)
	return true
}

// Error is used to report an error
func (r *CapturingReporterV2) Error(err error) bool {
	r.errs = append(r.errs, err)
	return true
}

// GetEvents returns all reported events
func (r *CapturingReporterV2) GetEvents() []mb.Event {
	return r.events
}

// GetErrors returns all reported errors
func (r *CapturingReporterV2) GetErrors() []error {
	return r.errs
}

// ReportingFetchV2 runs the given reporting metricset and returns all of the
// events and errors that occur during that period.
func ReportingFetchV2(metricSet mb.ReportingMetricSetV2) ([]mb.Event, []error) {
	r := &CapturingReporterV2{}
	metricSet.Fetch(r)
	return r.events, r.errs
}

// ReportingFetchV2Error runs the given reporting metricset and returns all of the
// events and errors that occur during that period.
func ReportingFetchV2Error(metricSet mb.ReportingMetricSetV2Error) ([]mb.Event, []error) {
	r := &CapturingReporterV2{}
	err := metricSet.Fetch(r)
	if err != nil {
		r.errs = append(r.errs, err)
	}
	return r.events, r.errs
}

// ReportingFetchV2WithContext runs the given reporting metricset and returns all of the
// events and errors that occur during that period.
func ReportingFetchV2WithContext(metricSet mb.ReportingMetricSetV2WithContext) ([]mb.Event, []error) {
	r := &CapturingReporterV2{}
	err := metricSet.Fetch(context.Background(), r)
	if err != nil {
		r.errs = append(r.errs, err)
	}
	return r.events, r.errs
}

// NewPushMetricSet instantiates a new PushMetricSet using the given
// configuration. The ModuleFactory and MetricSetFactory are obtained from the
// global Registry.
func NewPushMetricSet(t testing.TB, config interface{}) mb.PushMetricSet {
	metricSet := NewMetricSet(t, config)

	pushMetricSet, ok := metricSet.(mb.PushMetricSet)
	if !ok {
		t.Fatal("MetricSet does not implement PushMetricSet")
	}

	return pushMetricSet
}

type capturingReporter struct {
	events []common.MapStr
	errs   []error
	done   chan struct{}
}

func (r *capturingReporter) Event(event common.MapStr) bool {
	r.events = append(r.events, event)
	return true
}

func (r *capturingReporter) ErrorWith(err error, meta common.MapStr) bool {
	r.events = append(r.events, meta)
	r.errs = append(r.errs, err)
	return true
}

func (r *capturingReporter) Error(err error) bool {
	r.errs = append(r.errs, err)
	return true
}

func (r *capturingReporter) Done() <-chan struct{} {
	return r.done
}

// RunPushMetricSet run the given push metricset for the specific amount of time
// and returns all of the events and errors that occur during that period.
func RunPushMetricSet(duration time.Duration, metricSet mb.PushMetricSet) ([]common.MapStr, []error) {
	r := &capturingReporter{done: make(chan struct{})}

	// Run the metricset.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		metricSet.Run(r)
	}()

	// Let it run for some period, then stop it by closing the done channel.
	time.AfterFunc(duration, func() {
		close(r.done)
	})

	// Wait for the PushMetricSet to completely stop.
	wg.Wait()

	// Return all events and errors that were collected.
	return r.events, r.errs
}

// NewPushMetricSetV2 instantiates a new PushMetricSetV2 using the given
// configuration. The ModuleFactory and MetricSetFactory are obtained from the
// global Registry.
func NewPushMetricSetV2(t testing.TB, config interface{}) mb.PushMetricSetV2 {
	metricSet := NewMetricSet(t, config)

	pushMetricSet, ok := metricSet.(mb.PushMetricSetV2)
	if !ok {
		t.Fatal("MetricSet does not implement PushMetricSetV2")
	}

	return pushMetricSet
}

// NewPushMetricSetV2WithContext instantiates a new PushMetricSetV2WithContext
// using the given configuration. The ModuleFactory and MetricSetFactory are
// obtained from the global Registry.
func NewPushMetricSetV2WithContext(t testing.TB, config interface{}) mb.PushMetricSetV2WithContext {
	metricSet := NewMetricSet(t, config)

	pushMetricSet, ok := metricSet.(mb.PushMetricSetV2WithContext)
	if !ok {
		t.Fatal("MetricSet does not implement PushMetricSetV2WithContext")
	}

	return pushMetricSet
}

// capturingPushReporterV2 stores all the events and errors from a metricset's
// Run method.
type capturingPushReporterV2 struct {
	context.Context
	eventsC chan mb.Event
}

func newCapturingPushReporterV2(ctx context.Context) *capturingPushReporterV2 {
	return &capturingPushReporterV2{Context: ctx, eventsC: make(chan mb.Event)}
}

// report writes an event to the output channel and returns true. If the output
// is closed it returns false.
func (r *capturingPushReporterV2) report(event mb.Event) bool {
	select {
	case <-r.Done():
		// Publisher is stopped.
		return false
	case r.eventsC <- event:
		return true
	}
}

// Event stores the passed-in event into the events array
func (r *capturingPushReporterV2) Event(event mb.Event) bool {
	return r.report(event)
}

// Error stores the given error into the errors array.
func (r *capturingPushReporterV2) Error(err error) bool {
	return r.report(mb.Event{Error: err})
}

func (r *capturingPushReporterV2) capture(waitEvents int) []mb.Event {
	var events []mb.Event
	for {
		select {
		case <-r.Done():
			// Timeout
			return events
		case e := <-r.eventsC:
			events = append(events, e)
			if waitEvents > 0 && len(events) >= waitEvents {
				return events
			}
		}
	}
}

// RunPushMetricSetV2 run the given push metricset for the specific amount of
// time and returns all of the events and errors that occur during that period.
func RunPushMetricSetV2(timeout time.Duration, waitEvents int, metricSet mb.PushMetricSetV2) []mb.Event {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r := newCapturingPushReporterV2(ctx)

	go metricSet.Run(r)
	return r.capture(waitEvents)
}

// RunPushMetricSetV2WithContext run the given push metricset for the specific amount of
// time and returns all of the events that occur during that period.
func RunPushMetricSetV2WithContext(timeout time.Duration, waitEvents int, metricSet mb.PushMetricSetV2WithContext) []mb.Event {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r := newCapturingPushReporterV2(ctx)

	go metricSet.Run(ctx, r)
	return r.capture(waitEvents)
}
