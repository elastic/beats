// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package application_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

// TestNewReaderValid should successfully instantiate the reader.
func TestNewReaderValid(t *testing.T) {
	var config Config
	reader, err := newReader(config, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.NotNil(t, reader.query)
	assert.NotNil(t, reader.query.Handle)
	assert.NotNil(t, reader.query.Counters)
	defer reader.close()
}

// TestInitCounters should successfully instantiate the reader counters.
func TestInitCounters(t *testing.T) {
	var config Config
	reader, err := newReader(config, logptest.NewTestingLogger(t, ""))
	assert.NotNil(t, reader)
	assert.NoError(t, err)
	// if iis is not enabled, the reader.ApplicationPools is empty
	if len(reader.applicationPools) > 0 {
		assert.NotZero(t, len(reader.query.Counters))
		assert.NotZero(t, len(reader.workerProcesses))
	}
	defer reader.close()
}

func TestGetProcessIds(t *testing.T) {
	tests := map[string]struct {
		counterValues map[string][]pdh.CounterValue
		counterFields map[string]string
	}{
		"english": {
			counterValues: map[string][]pdh.CounterValue{
				"\\Process(w3wp#1)\\ID Process":   {{Instance: "w3wp#1", Measurement: 124.00}},
				"\\Process(w3wp#1)\\Thread Count": {{Instance: "w3wp#1", Measurement: 30.00}},
			},
			counterFields: map[string]string{
				"\\Process(w3wp#1)\\ID Process":   ecsProcessId,
				"\\Process(w3wp#1)\\Thread Count": "process.thread_count",
			},
		},
		// On non-English Windows the expanded counter paths are localized.
		"german": {
			counterValues: map[string][]pdh.CounterValue{
				"\\Prozess(w3wp#1)\\Prozesskennung": {{Instance: "w3wp#1", Measurement: 124.00}},
				"\\Prozess(w3wp#1)\\Threadanzahl":   {{Instance: "w3wp#1", Measurement: 30.00}},
			},
			counterFields: map[string]string{
				"\\Prozess(w3wp#1)\\Prozesskennung": ecsProcessId,
				"\\Prozess(w3wp#1)\\Threadanzahl":   "process.thread_count",
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			workerProcesses := getProcessIds(tc.counterValues, tc.counterFields)
			if !assert.Len(t, workerProcesses, 1, "expected exactly one worker process from the PID counter") {
				return
			}
			assert.Equal(t, 124, workerProcesses[0].processId, "worker process ID should come from the PID counter value")
			assert.Equal(t, "w3wp#1", workerProcesses[0].instanceName, "worker instance name should come from the PID counter instance")
		})
	}
}
