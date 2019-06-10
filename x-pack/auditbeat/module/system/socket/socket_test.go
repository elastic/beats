// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import (
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	abtest "github.com/elastic/beats/auditbeat/testing"
	"github.com/elastic/beats/libbeat/common"
	sock "github.com/elastic/beats/metricbeat/helper/socket"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/gosigar/sys/linux"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())

	// Set lastState and add test process to cache so it will be reported as stopped.
	f.(*MetricSet).lastState = time.Now()
	s := testSocket()
	f.(*MetricSet).cache.DiffAndUpdateCache(convertToCacheable([]*Socket{s}))

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	// The first socket (events[0]) is usually something like rpcbind,
	// the last one should be more interesting.
	fullEvent := mbtest.StandardizeEvent(f, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestSocket(t *testing.T) {
	s := testSocket()

	assert.Equal(t, uint64(0xee1186910755e9b1), s.Hash())
	assert.Equal(t, "fIj66YRoGyoe8dML", s.entityID("fa8a1edd06864f47ba4cad5d0f5ca134"))
	assert.Equal(t, "1:IXrg9Y06W7zrkqBlE30jpC/mzjo=", s.communityID())
}

func testSocket() *Socket {
	return &Socket{
		Family:      linux.AF_INET,
		Protocol:    tcp,
		LocalIP:     net.IPv4(10, 0, 2, 15),
		LocalPort:   22,
		RemoteIP:    net.IPv4(10, 0, 2, 2),
		RemotePort:  55270,
		Direction:   sock.Inbound,
		UID:         0,
		Username:    "root",
		ProcessPID:  22799,
		ProcessName: "sshd",
	}
}

func TestOutbound(t *testing.T) {
	ms := mbtest.NewReportingMetricSetV2(t, getConfig())

	// Consume first set of events - list of all currently open sockets
	events, errs := mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	conn, err := net.Dial("tcp", "google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	localPort := getPort(t, conn.LocalAddr())

	// Consume second set of events - should contain socket we just opened
	events, errs = mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	var event *mb.Event
	for _, evt := range events {
		sourcePort, err := evt.RootFields.GetValue("source.port")
		if assert.NoError(t, err) {
			if sourcePort == localPort {
				event = &evt
				break
			}
		}
	}

	if event == nil {
		t.Fatal("socket not found")
	}

	checkFieldValue(t, event.RootFields, "event.action", eventActionSocketOpened.String())
	checkFieldValue(t, event.RootFields, "process.pid", os.Getpid())
	checkFieldValue(t, event.RootFields, "process.name", "socket.test")
	checkFieldValue(t, event.RootFields, "user.id", os.Geteuid())
	checkFieldValue(t, event.RootFields, "network.direction", sock.Outbound.String())
	checkFieldValue(t, event.RootFields, "network.transport", "tcp")
	checkFieldValue(t, event.RootFields, "destination.port", 80)

	communityID, err := event.RootFields.GetValue("network.community_id")
	if assert.NoError(t, err) {
		assert.NotEmpty(t, communityID)
	}
}

func TestListening(t *testing.T) {
	ms := mbtest.NewReportingMetricSetV2(t, getConfig())

	// Consume first set of events - list of all currently open sockets
	events, errs := mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	listenerPort := getPort(t, ln.Addr())

	// Consume second set of events - should contain socket we just opened
	events, errs = mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	var event *mb.Event
	for _, evt := range events {
		destinationPort, err := evt.RootFields.GetValue("destination.port")
		if assert.NoError(t, err) {
			if destinationPort == listenerPort {
				event = &evt
				break
			}
		}
	}

	if event == nil {
		t.Fatal("socket not found")
	}

	checkFieldValue(t, event.RootFields, "event.action", eventActionSocketOpened.String())
	checkFieldValue(t, event.RootFields, "process.pid", os.Getpid())
	checkFieldValue(t, event.RootFields, "process.name", "socket.test")
	checkFieldValue(t, event.RootFields, "user.id", os.Geteuid())
	checkFieldValue(t, event.RootFields, "network.direction", sock.Listening.String())
	checkFieldValue(t, event.RootFields, "network.transport", "tcp")
}

func TestLocalhost(t *testing.T) {
	config := getConfig()
	config["socket.include_localhost"] = true

	ms := mbtest.NewReportingMetricSetV2(t, config)

	// Consume first set of events - list of all currently open sockets
	events, errs := mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	listenerPort := getPort(t, ln.Addr())

	events, errs = mbtest.ReportingFetchV2(ms)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	var event *mb.Event
	for _, evt := range events {
		destinationPort, err := evt.RootFields.GetValue("destination.port")
		if assert.NoError(t, err) {
			if destinationPort == listenerPort {
				event = &evt
				break
			}
		}
	}

	if event == nil {
		t.Fatal("socket not found")
	}

	checkFieldValue(t, event.RootFields, "event.action", eventActionSocketOpened.String())
	checkFieldValue(t, event.RootFields, "process.pid", os.Getpid())
	checkFieldValue(t, event.RootFields, "process.name", "socket.test")
	checkFieldValue(t, event.RootFields, "user.id", os.Geteuid())
	checkFieldValue(t, event.RootFields, "network.direction", sock.Listening.String())
	checkFieldValue(t, event.RootFields, "network.transport", "tcp")
	checkFieldValue(t, event.RootFields, "destination.ip", "127.0.0.1")
}

func TestLocalhostExcluded(t *testing.T) {
	config := getConfig()
	config["socket.include_localhost"] = false

	ms := mbtest.NewReportingMetricSetV2(t, config)

	ln, err := net.Listen("tcp4", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	listenerPort := getPort(t, ln.Addr())

	events, errs := mbtest.ReportingFetchV2(ms)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	var event *mb.Event
	for _, evt := range events {
		destinationPort, err := evt.RootFields.GetValue("destination.port")
		if assert.NoError(t, err) {
			if destinationPort == listenerPort {
				event = &evt
				break
			}
		}
	}

	if event != nil {
		t.Fatalf("unexpected socket found: %v", event)
	}
}

func checkFieldValue(t *testing.T, mapstr common.MapStr, fieldName string, fieldValue interface{}) {
	value, err := mapstr.GetValue(fieldName)
	if assert.NoError(t, err) {
		switch v := value.(type) {
		case uint32:
			assert.Equal(t, fieldValue, int(v))
		case net.IP:
			assert.Equal(t, fieldValue, v.String())
		default:
			assert.Equal(t, fieldValue, v)
		}
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   "system",
		"datasets": []string{"socket"},
	}
}

func getPort(t *testing.T, addr net.Addr) int {
	s := addr.String()
	i := strings.LastIndex(s, ":")

	port, err := strconv.Atoi(s[i+1:])
	if err != nil {
		t.Fatal("failed to get port from addr", addr)
	}

	return port
}
