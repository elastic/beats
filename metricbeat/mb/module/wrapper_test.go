// +build !integration

package module_test

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"

	"github.com/stretchr/testify/assert"
)

const (
	moduleName           = "fake"
	eventFetcherName     = "EventFetcher"
	reportingFetcherName = "ReportingFetcher"
	pushMetricSetName    = "PushMetricSet"
)

// fakeMetricSet

func init() {
	if err := mb.Registry.AddMetricSet(moduleName, eventFetcherName, newFakeEventFetcher); err != nil {
		panic(err)
	}
	if err := mb.Registry.AddMetricSet(moduleName, reportingFetcherName, newFakeReportingFetcher); err != nil {
		panic(err)
	}
	if err := mb.Registry.AddMetricSet(moduleName, pushMetricSetName, newFakePushMetricSet); err != nil {
		panic(err)
	}
}

// EventFetcher

type fakeEventFetcher struct {
	mb.BaseMetricSet
}

func (ms *fakeEventFetcher) Fetch() (common.MapStr, error) {
	t, _ := time.Parse(time.RFC3339, "2016-05-10T23:27:58.485Z")
	return common.MapStr{"@timestamp": common.Time(t), "metric": 1}, nil
}

func (ms *fakeEventFetcher) Close() error {
	return nil
}

func newFakeEventFetcher(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &fakeEventFetcher{BaseMetricSet: base}, nil
}

// ReportingFetcher

type fakeReportingFetcher struct {
	mb.BaseMetricSet
}

func (ms *fakeReportingFetcher) Fetch(r mb.Reporter) {
	t, _ := time.Parse(time.RFC3339, "2016-05-10T23:27:58.485Z")
	r.Event(common.MapStr{"@timestamp": common.Time(t), "metric": 1})
}

func newFakeReportingFetcher(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &fakeReportingFetcher{BaseMetricSet: base}, nil
}

// PushMetricSet

type fakePushMetricSet struct {
	mb.BaseMetricSet
}

func (ms *fakePushMetricSet) Run(r mb.PushReporter) {
	t, _ := time.Parse(time.RFC3339, "2016-05-10T23:27:58.485Z")
	event := common.MapStr{"@timestamp": common.Time(t), "metric": 1}
	r.Event(event)
	<-r.Done()
}

func newFakePushMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &fakePushMetricSet{BaseMetricSet: base}, nil
}

// test utilities

func newTestRegistry(t testing.TB) *mb.Register {
	r := mb.NewRegister()

	if err := r.AddMetricSet(moduleName, eventFetcherName, newFakeEventFetcher); err != nil {
		t.Fatal(err)
	}
	if err := r.AddMetricSet(moduleName, reportingFetcherName, newFakeReportingFetcher); err != nil {
		t.Fatal(err)
	}
	if err := r.AddMetricSet(moduleName, pushMetricSetName, newFakePushMetricSet); err != nil {
		t.Fatal(err)
	}

	return r
}

func newConfig(t testing.TB, moduleConfig interface{}) *common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

// test cases

func TestWrapperOfEventFetcher(t *testing.T) {
	hosts := []string{"alpha", "beta"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName},
		"hosts":      hosts,
	})

	m, err := module.NewWrapper(c, newTestRegistry(t))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	output := m.Start(done)

	<-output
	<-output
	close(done)

	// Validate that the channel is closed after receiving the two
	// initial events.
	select {
	case _, ok := <-output:
		if !ok {
			// Channel is closed.
			return
		} else {
			assert.Fail(t, "received unexpected event")
		}
	}
}

func TestWrapperOfReportingFetcher(t *testing.T) {
	hosts := []string{"alpha", "beta"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"hosts":      hosts,
	})

	m, err := module.NewWrapper(c, newTestRegistry(t))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	output := m.Start(done)

	<-output
	<-output
	close(done)

	// Validate that the channel is closed after receiving the two
	// initial events.
	select {
	case _, ok := <-output:
		if !ok {
			// Channel is closed.
			return
		} else {
			assert.Fail(t, "received unexpected event")
		}
	}
}

func TestWrapperOfPushMetricSet(t *testing.T) {
	hosts := []string{"alpha"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{pushMetricSetName},
		"hosts":      hosts,
	})

	m, err := module.NewWrapper(c, newTestRegistry(t))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	output := m.Start(done)

	<-output
	close(done)

	// Validate that the channel is closed after receiving the two
	// initial events.
	select {
	case _, ok := <-output:
		if !ok {
			// Channel is closed.
			return
		} else {
			assert.Fail(t, "received unexpected event")
		}
	}
}
