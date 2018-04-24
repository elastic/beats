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

// newMetricSet instantiates a new MetricSet using the given configuration.
// The ModuleFactory and MetricSetFactory are obtained from the global
// Registry.
func newMetricSet(t testing.TB, config interface{}) mb.MetricSet {
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
	metricSet := newMetricSet(t, config)

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
	metricSet := newMetricSet(t, config)

	fetcher, ok := metricSet.(mb.EventsFetcher)
	if !ok {
		t.Fatal("MetricSet does not implement EventsFetcher")
	}

	return fetcher
}

func NewReportingMetricSet(t testing.TB, config interface{}) mb.ReportingMetricSet {
	metricSet := newMetricSet(t, config)

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
	metricSet := newMetricSet(t, config)

	reportingMetricSetV2, ok := metricSet.(mb.ReportingMetricSetV2)
	if !ok {
		t.Fatal("MetricSet does not implement ReportingMetricSetV2")
	}

	return reportingMetricSetV2
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

// NewPushMetricSet instantiates a new PushMetricSet using the given
// configuration. The ModuleFactory and MetricSetFactory are obtained from the
// global Registry.
func NewPushMetricSet(t testing.TB, config interface{}) mb.PushMetricSet {
	metricSet := newMetricSet(t, config)

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
	metricSet := newMetricSet(t, config)

	pushMetricSet, ok := metricSet.(mb.PushMetricSetV2)
	if !ok {
		t.Fatal("MetricSet does not implement PushMetricSet")
	}

	return pushMetricSet
}

// capturingPushReporterV2 stores all the events and errors from a metricset's
// Run method.
type capturingPushReporterV2 struct {
	doneC   chan struct{}
	eventsC chan mb.Event
}

// report writes an event to the output channel and returns true. If the output
// is closed it returns false.
func (r *capturingPushReporterV2) report(event mb.Event) bool {
	select {
	case <-r.doneC:
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

// Done returns the Done channel for this reporter.
func (r *capturingPushReporterV2) Done() <-chan struct{} {
	return r.doneC
}

// RunPushMetricSetV2 run the given push metricset for the specific amount of
// time and returns all of the events and errors that occur during that period.
func RunPushMetricSetV2(timeout time.Duration, waitEvents int, metricSet mb.PushMetricSetV2) []mb.Event {
	var (
		r      = &capturingPushReporterV2{doneC: make(chan struct{}), eventsC: make(chan mb.Event)}
		wg     sync.WaitGroup
		events []mb.Event
	)
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		defer close(r.eventsC)
		if closer, ok := metricSet.(mb.Closer); ok {
			defer closer.Close()
		}
		metricSet.Run(r)
	}()

	// Consumer
	go func() {
		defer wg.Done()
		defer close(r.doneC)

		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				return
			case e, ok := <-r.eventsC:
				if !ok {
					return
				}
				events = append(events, e)
				if waitEvents > 0 && waitEvents <= len(events) {
					return
				}
			}
		}
	}()

	wg.Wait()
	return events
}
