package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/mapscheme"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func executeHTTPMonitorHostJob(t *testing.T, handlerFunc http.HandlerFunc) (*httptest.Server, beat.Event) {
	server := httptest.NewServer(handlerFunc)
	defer server.Close()

	config := common.NewConfig()
	config.SetString("urls", 0, server.URL)

	jobs, err := create(monitors.Info{}, config)
	if err != nil {
		t.FailNow()
	}
	job := jobs[0]

	event, _, err := job.Run()

	return server, event
}

func httpChecks(urlStr string, statusCode int) mapscheme.MapCheckDef {
	return mapscheme.MapCheckDef{
		"http": mapscheme.MapCheckDef{
			"url": urlStr,
			"response.status_code":   statusCode,
			"rtt.content.us":         mapscheme.IsDuration,
			"rtt.response_header.us": mapscheme.IsDuration,
			"rtt.total.us":           mapscheme.IsDuration,
			"rtt.validate.us":        mapscheme.IsDuration,
			"rtt.write_request.us":   mapscheme.IsDuration,
		},
	}
}

func httpErrorChecks(urlStr string, statusCode int) mapscheme.MapCheckDef {
	return mapscheme.MapCheckDef{
		"error": mapscheme.MapCheckDef{
			"message": "502 Bad Gateway",
			"type":    "validate",
		},
		"http": mapscheme.MapCheckDef{
			"url": urlStr,
			// TODO: This should work in the future "response.status_code":   statusCode,
			"rtt.content.us":         mapscheme.IsDuration,
			"rtt.response_header.us": mapscheme.IsDuration,
			"rtt.validate.us":        mapscheme.IsDuration,
			"rtt.write_request.us":   mapscheme.IsDuration,
		},
	}
}

func TestOKJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, mapscheme.HelloWorldHandler)
	port, err := mapscheme.ServerPort(server)
	assert.Nil(t, err)

	mapscheme.Validate(t, mapscheme.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "up"), event.Fields)
	mapscheme.Validate(t, mapscheme.TcpChecks(port), event.Fields)
	mapscheme.Validate(t, httpChecks(server.URL, http.StatusOK), event.Fields)
}

func TestBadGatewayJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, mapscheme.BadGatewayHandler)
	port, err := mapscheme.ServerPort(server)
	assert.Nil(t, err)

	mapscheme.Validate(t, mapscheme.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "down"), event.Fields)
	mapscheme.Validate(t, mapscheme.TcpChecks(port), event.Fields)
	mapscheme.Validate(t, httpErrorChecks(server.URL, http.StatusBadGateway), event.Fields)
}
