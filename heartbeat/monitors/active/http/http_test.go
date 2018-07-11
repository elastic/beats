package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/testcommon"
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

func httpChecks(urlStr string, statusCode int) testcommon.MapCheckDef {
	return testcommon.MapCheckDef{
		"http": testcommon.MapCheckDef{
			"url": urlStr,
			"response.status_code":   statusCode,
			"rtt.content.us":         testcommon.IsDuration,
			"rtt.response_header.us": testcommon.IsDuration,
			"rtt.total.us":           testcommon.IsDuration,
			"rtt.validate.us":        testcommon.IsDuration,
			"rtt.write_request.us":   testcommon.IsDuration,
		},
	}
}

func httpErrorChecks(urlStr string, statusCode int) testcommon.MapCheckDef {
	return testcommon.MapCheckDef{
		"error": testcommon.MapCheckDef{
			"message": "502 Bad Gateway",
			"type":    "validate",
		},
		"http": testcommon.MapCheckDef{
			"url":                    urlStr,
			"rtt.content.us":         testcommon.IsDuration,
			"rtt.response_header.us": testcommon.IsDuration,
			"rtt.validate.us":        testcommon.IsDuration,
			"rtt.write_request.us":   testcommon.IsDuration,
		},
	}
}

func TestOKJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, testcommon.HelloWorldHandler)
	port, err := testcommon.ServerPort(server)
	assert.Nil(t, err)

	testcommon.DeepMapStrCheck(t, testcommon.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "up"), event.Fields)
	testcommon.DeepMapStrCheck(t, testcommon.TcpChecks(port), event.Fields)
	testcommon.DeepMapStrCheck(t, httpChecks(server.URL, http.StatusOK), event.Fields)
}

func TestBadGatewayJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, testcommon.BadGatewayHandler)
	port, err := testcommon.ServerPort(server)
	assert.Nil(t, err)

	testcommon.DeepMapStrCheck(t, testcommon.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "down"), event.Fields)
	testcommon.DeepMapStrCheck(t, testcommon.TcpChecks(port), event.Fields)
	testcommon.DeepMapStrCheck(t, httpErrorChecks(server.URL, http.StatusBadGateway), event.Fields)
}
