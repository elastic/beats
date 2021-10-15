// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cloudfoundry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

func TestGetSelectors(t *testing.T) {

	tests := []struct {
		Name      string
		Callbacks RlpListenerCallbacks
		Selectors []*loggregator_v2.Selector
	}{
		{
			Name: "HTTPAccess only",
			Callbacks: RlpListenerCallbacks{
				HttpAccess: func(_ *EventHttpAccess) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Timer{
						Timer: &loggregator_v2.TimerSelector{},
					},
				},
			},
		},
		{
			Name: "Log only",
			Callbacks: RlpListenerCallbacks{
				Log: func(_ *EventLog) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Log{
						Log: &loggregator_v2.LogSelector{},
					},
				},
			},
		},
		{
			Name: "Counter only",
			Callbacks: RlpListenerCallbacks{
				Counter: func(_ *EventCounter) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Counter{
						Counter: &loggregator_v2.CounterSelector{},
					},
				},
			},
		},
		{
			Name: "ValueMetric only",
			Callbacks: RlpListenerCallbacks{
				ValueMetric: func(_ *EventValueMetric) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Gauge{
						Gauge: &loggregator_v2.GaugeSelector{},
					},
				},
			},
		},
		{
			Name: "ContainerMetric only",
			Callbacks: RlpListenerCallbacks{
				ContainerMetric: func(_ *EventContainerMetric) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Gauge{
						Gauge: &loggregator_v2.GaugeSelector{},
					},
				},
			},
		},
		{
			Name: "Error only",
			Callbacks: RlpListenerCallbacks{
				Error: func(_ *EventError) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Event{
						Event: &loggregator_v2.EventSelector{},
					},
				},
			},
		},
		{
			Name: "ValueMetric and ContainerMetric",
			Callbacks: RlpListenerCallbacks{
				ValueMetric:     func(_ *EventValueMetric) {},
				ContainerMetric: func(_ *EventContainerMetric) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Gauge{
						Gauge: &loggregator_v2.GaugeSelector{},
					},
				},
			},
		},
		{
			Name: "All",
			Callbacks: RlpListenerCallbacks{
				HttpAccess:      func(_ *EventHttpAccess) {},
				Log:             func(_ *EventLog) {},
				Counter:         func(_ *EventCounter) {},
				ValueMetric:     func(_ *EventValueMetric) {},
				ContainerMetric: func(_ *EventContainerMetric) {},
				Error:           func(_ *EventError) {},
			},
			Selectors: []*loggregator_v2.Selector{
				{
					Message: &loggregator_v2.Selector_Timer{
						Timer: &loggregator_v2.TimerSelector{},
					},
				},
				{
					Message: &loggregator_v2.Selector_Log{
						Log: &loggregator_v2.LogSelector{},
					},
				},
				{
					Message: &loggregator_v2.Selector_Counter{
						Counter: &loggregator_v2.CounterSelector{},
					},
				},
				{
					Message: &loggregator_v2.Selector_Gauge{
						Gauge: &loggregator_v2.GaugeSelector{},
					},
				},
				{
					Message: &loggregator_v2.Selector_Event{
						Event: &loggregator_v2.EventSelector{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			listener := newRlpListener("", nil, "", test.Callbacks, nil)
			observed := listener.getSelectors()
			assert.EqualValues(t, test.Selectors, observed)
		})
	}

}
