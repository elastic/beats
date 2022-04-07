// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package application_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/metricbeat/helper/windows/pdh"
)

// TestNewReaderValid should successfully instantiate the reader.
func TestNewReaderValid(t *testing.T) {
	var config Config
	reader, err := newReader(config)
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
	reader, err := newReader(config)
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
	var key = "\\Process(w3wp#1)\\ID Process"
	var counters = []pdh.CounterValue{
		{
			Instance:    "w3wp#1",
			Measurement: 124.00,
			Err:         pdh.CounterValueError{},
		},
	}
	counterList := make(map[string][]pdh.CounterValue)
	counterList[key] = counters
	workerProcesses := getProcessIds(counterList)
	assert.NotZero(t, len(workerProcesses))
	assert.Equal(t, float64(workerProcesses[0].processId), counters[0].Measurement.(float64))
	assert.Equal(t, workerProcesses[0].instanceName, counters[0].Instance)
}
