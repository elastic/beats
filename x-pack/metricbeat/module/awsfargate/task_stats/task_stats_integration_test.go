// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package task_stats

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/mb/testing/flags"
)

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

	taskStatsFile, err := os.Open("./_meta/testdata/task_stats.json")
	assert.NoError(t, err)
	defer taskStatsFile.Close()

	byteTaskStats, err := ioutil.ReadAll(taskStatsFile)
	assert.NoError(t, err)

	taskStatsResp := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(byteTaskStats)),
	}

	taskFile, err := os.Open("./_meta/testdata/task.json")
	assert.NoError(t, err)
	defer taskStatsFile.Close()

	byteTask, err := ioutil.ReadAll(taskFile)
	assert.NoError(t, err)

	byteTaskResp := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(byteTask)),
	}

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)

	taskOutput, err := getTask(byteTaskResp)
	assert.NoError(t, err)

	formattedStats := getStatsList(taskStatsOutput, taskOutput)
	assert.Equal(t, 1, len(formattedStats))
	event := createEvent(&formattedStats[0])
	standardizeEvent := m.StandardizeEvent(event)

	mbtest.WriteEventToDataJSON(t, standardizeEvent, "")
}
