// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package statsd

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/auditbeat/core"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/statsd/server"
)

func init() {
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

const (
	STATSD_HOST = "localhost"
	STATSD_PORT = 8126
)

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "airflow",
		"metricsets": []string{"statsd"},
		"host":       STATSD_HOST,
		"port":       STATSD_PORT,
		"period":     "100ms",
		"ttl":        "1ms",
	}
}

func createEvent(data string, t *testing.T) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", STATSD_HOST, STATSD_PORT))
	require.NoError(t, err)

	conn, err := net.DialUDP("udp", nil, udpAddr)
	require.NoError(t, err)

	_, err = fmt.Fprint(conn, data)
	require.NoError(t, err)
}

func TestData(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping `data.json` generation test")
	}

	ms := mbtest.NewPushMetricSetV2(t, getConfig())
	var events []mb.Event
	var reporter mb.PushReporterV2
	done := make(chan interface{})
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		reporter = mbtest.GetCapturingPushReporterV2()
		ms.(*server.MetricSet).ServerStart()
		wg.Done()

		go ms.Run(reporter)
		events = reporter.(*mbtest.CapturingPushReporterV2).BlockingCapture(2)

		close(done)
	}(wg)

	wg.Wait()
	createEvent("dagrun.duration.failed.a_dagid:200|ms|#k1:v1,k2:v2", t)
	createEvent("dagrun.duration.failed.b_dagid:500|ms|#k1:v1,k2:v2", t)
	<-done
	assert.Len(t, events, 2)
	if len(events) == 0 {
		t.Fatal("received no events")
	}

	beatEvent := mbtest.StandardizeEvent(ms, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}
