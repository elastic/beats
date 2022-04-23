// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"strconv"

	"github.com/elastic/beats/v7/heartbeat/hbregistry"
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

var globalBrowserRecorder *BrowserStats = nil

type BrowserStats struct {
	stepsHistogram *monitoring.UniqueList // histogram with the count for monitors with each number of steps
}

func GetBrowserStats() *BrowserStats {
	if globalBrowserRecorder != nil {
		return globalBrowserRecorder
	}

	tr := hbregistry.StatsRegistry
	r := tr.GetRegistry("browser")

	stepsHistogram := monitoring.NewUniqueList()
	monitoring.NewFunc(r, "steps_histogram", stepsHistogram.ReportMap, monitoring.Report)

	s := BrowserStats{stepsHistogram}

	globalBrowserRecorder = &s

	return globalBrowserRecorder
}

func (b BrowserStats) RegisterStepCount(c int) {
	b.stepsHistogram.Add(strconv.Itoa(c))
}
