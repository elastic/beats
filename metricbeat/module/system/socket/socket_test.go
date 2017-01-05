// +build linux

package socket

import (
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestData(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	f := mbtest.NewEventsFetcher(t, getConfig())

	if err = mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
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

	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if err != nil {
		t.Fatal("fetch", err)
	}

	var found bool
	for _, evt := range events {
		port, ok := getRequiredValue("local.port", evt, t).(int)
		if !ok {
			t.Fatal("local.port is not an int")
		}
		if port != listenerPort {
			continue
		}

		pid, ok := getRequiredValue("process.pid", evt, t).(int)
		if !ok {
			t.Fatal("proess.pid is not a int")
		}
		assert.Equal(t, os.Getpid(), pid)

		uid, ok := getRequiredValue("user.id", evt, t).(uint32)
		if !ok {
			t.Fatal("user.id is not an uint32")
		}
		assert.EqualValues(t, os.Geteuid(), uid)

		dir, ok := getRequiredValue("direction", evt, t).(string)
		if !ok {
			t.Fatal("direction is not a string")
		}
		assert.Equal(t, "listening", dir)

		_ = getRequiredValue("process.cmdline", evt, t).(string)
		_ = getRequiredValue("process.command", evt, t).(string)
		_ = getRequiredValue("process.exe", evt, t).(string)

		found = true
		break
	}

	assert.True(t, found, "listener not found")
}

func getRequiredValue(key string, m common.MapStr, t testing.TB) interface{} {
	v, err := m.GetValue(key)
	if err != nil {
		t.Fatal(err)
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
