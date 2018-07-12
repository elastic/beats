package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/skima"
	"github.com/elastic/beats/heartbeat/testutil"
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

func httpChecks(urlStr string, statusCode int) skima.Validator {
	return skima.Schema(skima.Map{
		"http": skima.Map{
			"url": urlStr,
			"response.status_code":   statusCode,
			"rtt.content.us":         skima.IsDuration,
			"rtt.response_header.us": skima.IsDuration,
			"rtt.total.us":           skima.IsDuration,
			"rtt.validate.us":        skima.IsDuration,
			"rtt.write_request.us":   skima.IsDuration,
		},
	})
}

func httpErrorChecks(urlStr string, statusCode int) skima.Validator {
	return skima.Schema(skima.Map{
		"error": skima.Map{
			"message": "502 Bad Gateway",
			"type":    "validate",
		},
		"http": skima.Map{
			"url": urlStr,
			// TODO: This should work in the future "response.status_code":   statusCode,
			"rtt.content.us":         skima.IsDuration,
			"rtt.response_header.us": skima.IsDuration,
			"rtt.validate.us":        skima.IsDuration,
			"rtt.write_request.us":   skima.IsDuration,
		},
	})
}

func TestOKJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, testutil.HelloWorldHandler)
	port, err := testutil.ServerPort(server)
	assert.Nil(t, err)

	skima.Strict(skima.Compose(
		testutil.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "up"),
		testutil.TcpChecks(port),
		httpChecks(server.URL, http.StatusOK),
	))(t, event.Fields)
}

func TestBadGatewayJob(t *testing.T) {
	server, event := executeHTTPMonitorHostJob(t, testutil.BadGatewayHandler)
	port, err := testutil.ServerPort(server)
	assert.Nil(t, err)

	skima.Strict(skima.Compose(
		testutil.MonitorChecks("http@"+server.URL, "127.0.0.1", "http", "down"),
		testutil.TcpChecks(port),
		httpErrorChecks(server.URL, http.StatusBadGateway),
	))(t, event.Fields)
}
