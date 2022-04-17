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

//go:build linux
// +build linux

package socket

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
	sock "github.com/menderesk/beats/v7/metricbeat/helper/socket"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	_ "github.com/menderesk/beats/v7/metricbeat/module/system"
)

func TestData(t *testing.T) {
	directionIs := func(direction string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("network.direction")
			return err == nil && v == direction
		}
	}

	dataFiles := []struct {
		direction string
		path      string
	}{
		{sock.ListeningName, "."},
		{sock.IngressName, "./_meta/data_ingress.json"},
		{sock.EgressName, "./_meta/data_egress.json"},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	for _, df := range dataFiles {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		c.Close()

		t.Run(fmt.Sprintf("direction:%s", df.direction), func(t *testing.T) {
			err = mbtest.WriteEventsReporterV2ErrorCond(f, t, df.path, directionIs(df.direction))
			if err != nil {
				t.Fatal("write", err)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	i := strings.LastIndex(addr, ":")
	listenerPort, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		t.Fatal("failed to get port from addr", addr)
	}

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "socket").Fields.StringToPrint())

	var found bool
	for _, event := range events {
		root := event.BeatEvent("system", "socket").Fields

		s, err := root.GetValue("system.socket")
		require.NoError(t, err)

		fields, ok := s.(common.MapStr)
		require.True(t, ok)

		port, ok := getRequiredValue(t, "local.port", fields).(int)
		if !ok {
			t.Fatal("local.port is not an int")
		}
		if port != listenerPort {
			continue
		}

		pid, ok := getRequiredValue(t, "process.pid", root).(int)
		if !ok {
			t.Fatal("process.pid is not a int")
		}
		assert.Equal(t, os.Getpid(), pid)

		uid, ok := getRequiredValue(t, "user.id", root).(string)
		if !ok {
			t.Fatal("user.id is not a string")
		}
		assert.EqualValues(t, strconv.Itoa(os.Geteuid()), uid)

		dir, ok := getRequiredValue(t, "network.direction", root).(string)
		if !ok {
			t.Fatal("direction is not a string")
		}
		assert.Equal(t, "listening", dir)

		_ = getRequiredValue(t, "process.cmdline", fields).(string)
		_ = getRequiredValue(t, "process.name", root).(string)
		_ = getRequiredValue(t, "process.executable", root).(string)
		_ = getRequiredValue(t, "process.args", root).([]string)

		found = true
		break
	}

	assert.True(t, found, "listener not found")
}

func getRequiredValue(t testing.TB, key string, m common.MapStr) interface{} {
	v, err := m.GetValue(key)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get value for key '%s'", key))
	}
	if v == nil {
		t.Fatalf("key %v not found in %v", key, m)
	}
	return v
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"socket"},
	}
}
