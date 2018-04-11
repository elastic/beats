package syslog

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestWhenPriorityIsSet(t *testing.T) {
	e := newEvent()
	e.SetPriority([]byte("13"))
	e.SetMessage([]byte("hello world"))
	e.SetHostname([]byte("wopr"))
	e.SetPid([]byte("123"))

	m := dummyMetadata()
	event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))

	expected := common.MapStr{
		"source":   "127.0.0.1",
		"message":  "hello world",
		"hostname": "wopr",
		"process": common.MapStr{
			"pid": 123,
		},
		"event": common.MapStr{
			"severity": 5,
		},
		"syslog": common.MapStr{
			"facility":       1,
			"severity_label": "Notice",
			"facility_label": "user-level",
			"priority":       13,
		},
	}

	assert.Equal(t, expected, event.Fields)
}

func TestWhenPriorityIsNotSet(t *testing.T) {
	e := newEvent()
	e.SetMessage([]byte("hello world"))
	e.SetHostname([]byte("wopr"))
	e.SetPid([]byte("123"))

	m := dummyMetadata()
	event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))
	expected := common.MapStr{
		"source":   "127.0.0.1",
		"message":  "hello world",
		"hostname": "wopr",
		"process": common.MapStr{
			"pid": 123,
		},
		"event":  common.MapStr{},
		"syslog": common.MapStr{},
	}

	assert.Equal(t, expected, event.Fields)
}

func TestPid(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetPid([]byte("123"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))
		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, common.MapStr{"pid": 123}, v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))

		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, common.MapStr{}, v)
	})
}

func TestHostname(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetHostname([]byte("wopr"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))
		v, err := event.GetValue("hostname")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "wopr", v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))

		_, err := event.GetValue("hostname")
		if !assert.Error(t, err) {
			return
		}
	})
}

func TestProgram(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		e.SetProgram([]byte("sudo"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))
		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, common.MapStr{"program": "sudo"}, v)
	})

	t.Run("is not set", func(t *testing.T) {
		e := newEvent()
		e.SetMessage([]byte("hello world"))
		m := dummyMetadata()
		event := createEvent(e, m, time.Local, logp.NewLogger("syslog"))

		v, err := event.GetValue("process")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, common.MapStr{}, v)
	})
}

func dummyMetadata() inputsource.NetworkMetadata {
	ip := "127.0.0.1"
	parsedIP := net.ParseIP(ip)
	addr := &net.IPAddr{IP: parsedIP, Zone: ""}
	return inputsource.NetworkMetadata{RemoteAddr: addr}
}
