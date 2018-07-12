package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/valschema"
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

func httpChecks(urlStr string, statusCode int) valschema.Map {
	return valschema.Map{
		"http": valschema.Map{
			"url": urlStr,
			"response.status_code":   statusCode,
			"rtt.content.us":         valschema.IsDuration,
			"rtt.response_header.us": valschema.IsDuration,
			"rtt.total.us":           valschema.IsDuration,
			"rtt.validate.us":        valschema.IsDuration,
			"rtt.write_request.us":   valschema.IsDuration,
		},
	}
}

func httpErrorChecks(urlStr string, statusCode int) valschema.Map {
	return valschema.Map{
		"error": valschema.Map{
			"message": "502 Bad Gateway",
			"type":    "validate",
		},
		"http": valschema.Map{
			"url": urlStr,
			// TODO: This should work in the future "response.status_code":   statusCode,
			"rtt.content.us":         valschema.IsDuration,
			"rtt.response_header.us": valschema.IsDuration,
			"rtt.validate.us":        valschema.IsDuration,
			"rtt.write_request.us":   valschema.IsDuration,
		},
	}
}

func TestOKJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, valschema.HelloWorldHandler)
	port, err := valschema.ServerPort(server)
	assert.Nil(t, err)

	valschema.Validate(t, valschema.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "up"), event.Fields)
	valschema.Validate(t, valschema.TcpChecks(port), event.Fields)
	valschema.Validate(t, httpChecks(server.URL, http.StatusOK), event.Fields)
}

func TestBadGatewayJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, valschema.BadGatewayHandler)
	port, err := valschema.ServerPort(server)
	assert.Nil(t, err)

	valschema.Validate(t, valschema.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "down"), event.Fields)
	valschema.Validate(t, valschema.TcpChecks(port), event.Fields)
	valschema.Validate(t, httpErrorChecks(server.URL, http.StatusBadGateway), event.Fields)
}
