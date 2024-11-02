// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package task_stats

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/mb/testing/flags"
)

func TestFetch(t *testing.T) {
	config := map[string]interface{}{
		"period":     "10s",
		"module":     "awsfargate",
		"metricsets": []string{"task_stats"},
	}

	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", "1.2.3.4")

	taskStatsResp, err := buildResponse("./_meta/testdata/task_stats.json")
	assert.NoError(t, err)

	byteTaskResp, err := buildResponse("./_meta/testdata/task.json")
	assert.NoError(t, err)

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)

	taskOutput, err := getTask(byteTaskResp)
	assert.NoError(t, err)

	formattedStats := getStatsList(taskStatsOutput, taskOutput)
	assert.Equal(t, 1, len(formattedStats))
	event := createEvent(&formattedStats[0])

	// Build a metricset to test the event
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)

	// The goal here is to make sure every element inside the
	// event has a matching field ("no field left behind").
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, []mb.Event{event})

	t.Cleanup(func() {
		taskStatsResp.Body.Close()
		byteTaskResp.Body.Close()
	})
}

func TestData(t *testing.T) {
	if !*flags.DataFlag {
		t.Skip("skip data generation tests")
	}

	config := map[string]interface{}{
		"period":     "10s",
		"module":     "awsfargate",
		"metricsets": []string{"task_stats"},
	}

	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", "1.2.3.4")
	m := mbtest.NewFetcher(t, config)

	taskStatsResp, err := buildResponse("./_meta/testdata/task_stats.json")
	assert.NoError(t, err)

	byteTaskResp, err := buildResponse("./_meta/testdata/task.json")
	assert.NoError(t, err)

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)

	taskOutput, err := getTask(byteTaskResp)
	assert.NoError(t, err)

	formattedStats := getStatsList(taskStatsOutput, taskOutput)
	assert.Equal(t, 1, len(formattedStats))
	event := createEvent(&formattedStats[0])
	standardizeEvent := m.StandardizeEvent(event)

	mbtest.WriteEventToDataJSON(t, standardizeEvent, "")

	t.Cleanup(func() {
		taskStatsResp.Body.Close()
		byteTaskResp.Body.Close()
	})
}

// buildResponse is a test helper that loads the content of `filename` and returns
// it as the body of a `http.Response`.
func buildResponse(filename string) (*http.Response, error) {
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(fileContent)),
	}, nil
}
