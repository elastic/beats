// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package application_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
)

// TestNewReaderValid should successfully instantiate the reader.
func TestNewReaderValid(t *testing.T) {
	reader, err := newReader()
	assert.Nil(t, err)
	assert.NotNil(t, reader)
	assert.NotNil(t, reader.Query)
	assert.NotNil(t, reader.Query.Handle)
	assert.NotNil(t, reader.Query.Counters)
	assert.Zero(t, len(reader.Query.Counters))
	defer reader.close()
}

// TestInitCounters should successfully instantiate the reader counters.
func TestInitCounters(t *testing.T) {
	reader, err := newReader()
	assert.NotNil(t, reader)
	assert.Nil(t, err)

	err = reader.initCounters([]string{})
	assert.Nil(t, err)
	// if iis is not enabled, the reader.ApplicationPools is empty
	if len(reader.ApplicationPools) > 0 {
		assert.NotZero(t, len(reader.Query.Counters))
		assert.NotZero(t, len(reader.WorkerProcesses))
	}
	defer reader.close()
}

func TestGetProcessIds(t *testing.T) {
	var key = "\\Process(w3wp#1)\\ID Process"
	var counters = []pdh.CounterValue{
		{
			Instance:    "w3wp#1",
			Measurement: 124.00,
			Err:         nil,
		},
	}
	counterList := make(map[string][]pdh.CounterValue)
	counterList[key] = counters
	workerProcesses := getProcessIds(counterList)
	assert.NotZero(t, len(workerProcesses))
	assert.Equal(t, float64(workerProcesses[0].ProcessId), counters[0].Measurement.(float64))
	assert.Equal(t, workerProcesses[0].InstanceName, counters[0].Instance)
}
