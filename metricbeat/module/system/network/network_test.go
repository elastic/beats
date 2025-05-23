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

//go:build darwin || freebsd || linux || windows || aix

package network

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "network").Fields.StringToPrint())
}

func TestNormalHostMetrics(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test requires linux")
	}
	basePath, err := os.Getwd()
	require.NoError(t, err)

	reporter := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	firstProc := filepath.Join(basePath, "/tests/testdata/proc/")
	err = os.Setenv("HOST_PROC", firstProc)
	require.NoError(t, err)

	// get initial metrics
	_, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)

	// second event
	secondProc := filepath.Join(basePath, "/tests/testdata2/proc/")
	err = os.Setenv("HOST_PROC", secondProc)
	require.NoError(t, err)

	events, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)
	found, evt := findRootEvent(events)
	require.True(t, found)

	t.Logf("second event: %+v", evt.RootFields.StringToPrint())

	// check second values
	ingressBytes, err := evt.RootFields.GetValue("host.network.ingress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(110), ingressBytes)

	ingressPackets, err := evt.RootFields.GetValue("host.network.ingress.packets")
	require.NoError(t, err)
	require.Equal(t, uint64(200000), ingressPackets)

	egressPackets, err := evt.RootFields.GetValue("host.network.egress.packets")
	require.NoError(t, err)
	require.Equal(t, uint64(200001), egressPackets)

	egressBytes, err := evt.RootFields.GetValue("host.network.egress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(30100), egressBytes)
}

func TestRollover(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test requires linux")
	}
	basePath, err := os.Getwd()
	require.NoError(t, err)

	reporter := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	firstProc := filepath.Join(basePath, "/tests/rollover/proc/")
	err = os.Setenv("HOST_PROC", firstProc)
	require.NoError(t, err)

	_, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)

	secondProc := filepath.Join(basePath, "/tests/rollover2/proc/")
	err = os.Setenv("HOST_PROC", secondProc)
	require.NoError(t, err)

	events, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)
	found, evt := findRootEvent(events)
	require.True(t, found)

	t.Logf("second event: %+v", evt.RootFields.StringToPrint())

	ingressBytes, err := evt.RootFields.GetValue("host.network.ingress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(601), ingressBytes)

	egressBytes, err := evt.RootFields.GetValue("host.network.egress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(902), egressBytes)
}

func TestRollover32(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test requires linux")
	}
	basePath, err := os.Getwd()
	require.NoError(t, err)

	reporter := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	firstProc := filepath.Join(basePath, "/tests/rollover32/proc/")
	err = os.Setenv("HOST_PROC", firstProc)
	require.NoError(t, err)

	_, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)

	secondProc := filepath.Join(basePath, "/tests/rollover32_2/proc/")
	err = os.Setenv("HOST_PROC", secondProc)
	require.NoError(t, err)

	events, errs := mbtest.ReportingFetchV2Error(reporter)
	require.Empty(t, errs)
	found, evt := findRootEvent(events)
	require.True(t, found)

	t.Logf("second event: %+v", evt.RootFields.StringToPrint())

	egressBytes, err := evt.RootFields.GetValue("host.network.egress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(3037888886), egressBytes)

	ingressBytes, err := evt.RootFields.GetValue("host.network.ingress.bytes")
	require.NoError(t, err)
	require.Equal(t, uint64(1101), ingressBytes)
}

func TestGauge(t *testing.T) {
	var prevu32 uint64 = math.MaxUint32 - 10
	var currentu32 uint64 = 10

	resultu32 := createGaugeWithRollover(currentu32, prevu32)
	require.Equal(t, uint64(21), resultu32)

	var prevNoRollover uint64 = 347458374592
	var currentNoRollover = prevNoRollover + 3452
	resultNoRollover := createGaugeWithRollover(currentNoRollover, prevNoRollover)
	require.Equal(t, uint64(3452), resultNoRollover)

	var prevu64 uint64 = math.MaxUint64 - 5000
	var currentu64 uint64 = 32
	resultu64 := createGaugeWithRollover(currentu64, prevu64)
	require.Equal(t, uint64(5033), resultu64)
}

func TestGaugeRolloverIncrement(t *testing.T) {
	// test to see if we're properly incrementing when we rollover
	// i.e do we count the increment from MAX_INT to 0?
	var prevU64 uint64 = math.MaxUint64
	current := uint64(0)

	resultu32 := createGaugeWithRollover(current, prevU64)
	require.Equal(t, uint64(1), resultu32)
}

func findRootEvent(events []mb.Event) (bool, mb.Event) {
	for _, evt := range events {
		if len(evt.RootFields) > 0 {
			return true, evt
		}
	}
	return false, mb.Event{}
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"network"},
	}
}
