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

//go:build !integration
// +build !integration

package module_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/module"
	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	moduleName           = "fake"
	reportingFetcherName = "ReportingFetcher"
	pushMetricSetName    = "PushMetricSet"
)

// fakeMetricSet

func init() {
	mb.Registry.MustAddMetricSet(moduleName, reportingFetcherName, newFakeReportingFetcher)
	mb.Registry.MustAddMetricSet(moduleName, pushMetricSetName, newFakePushMetricSet)
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
	var r mb.ReportingMetricSet = &fakeReportingFetcher{BaseMetricSet: base}
	return r, nil
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
	var r mb.PushMetricSet = &fakePushMetricSet{BaseMetricSet: base}
	return r, nil
}

// test utilities

func newTestRegistry(t testing.TB) *mb.Register {
	r := mb.NewRegister()

	err := r.AddMetricSet(moduleName, reportingFetcherName, newFakeReportingFetcher)
	require.NoError(t, err)
	err = r.AddMetricSet(moduleName, pushMetricSetName, newFakePushMetricSet)
	require.NoError(t, err)
	return r
}

func newConfig(t testing.TB, moduleConfig interface{}) *conf.C {
	config, err := conf.NewConfigFrom(moduleConfig)
	require.NoError(t, err)
	return config
}

// test cases

func TestWrapperOfReportingFetcher(t *testing.T) {
	hosts := []string{"alpha", "beta"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"hosts":      hosts,
	})

	m, err := module.NewWrapper(c, newTestRegistry(t))
	require.NoError(t, err)

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
	require.NoError(t, err)

	done := make(chan struct{})
	output := m.Start(done)

	<-output
	close(done)

	// Validate that the channel is closed after receiving the event.
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

func TestPeriodIsAddedToEvent(t *testing.T) {
	cases := map[string]struct {
		metricset string
		hasPeriod bool
	}{
		"fetch metricset events should have period": {
			metricset: reportingFetcherName,
			hasPeriod: true,
		},
		"push metricset events should not have period": {
			metricset: pushMetricSetName,
			hasPeriod: false,
		},
	}

	registry := newTestRegistry(t)

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			hosts := []string{"alpha"}
			config := newConfig(t, map[string]interface{}{
				"module":     moduleName,
				"metricsets": []string{c.metricset},
				"hosts":      hosts,
			})

			m, err := module.NewWrapper(config, registry, module.WithMetricSetInfo())
			require.NoError(t, err)

			done := make(chan struct{})
			defer close(done)

			output := m.Start(done)

			event := <-output

			hasPeriod, _ := event.Fields.HasKey("metricset.period")
			assert.Equal(t, c.hasPeriod, hasPeriod, "has metricset.period in event %+v", event)
		})
	}
}

func TestNewWrapperForMetricSet(t *testing.T) {
	hosts := []string{"alpha"}
	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"hosts":      hosts,
	})

	aModule, metricSets, err := mb.NewModule(c, newTestRegistry(t))
	require.NoError(t, err)

	m, err := module.NewWrapperForMetricSet(aModule, metricSets[0], module.WithMetricSetInfo())
	require.NoError(t, err)

	done := make(chan struct{})
	output := m.Start(done)

	<-output
	close(done)

	// Validate that the channel is closed after receiving the event.
	select {
	case _, ok := <-output:
		if !ok {
			return // Channel is closed.
		}
		assert.Fail(t, "received unexpected event")
	}
}
