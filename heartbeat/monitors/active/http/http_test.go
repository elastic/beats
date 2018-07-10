package http

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

var helloWorldBody = "hello, world!"

var helloWorldHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, helloWorldBody)
})

var badGatewayBody = "Bad Gateway"

var badGatewayHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, badGatewayBody)
})

var exactlyEqual = func(expected interface{}) func(t *testing.T, actual interface{}) {
	return func(t *testing.T, actual interface{}) {
		assert.Equal(t, expected, actual)
	}
}

var isDuration = func(t *testing.T, actual interface{}) {
	converted, ok := actual.(time.Duration)
	assert.True(t, ok)
	assert.True(t, converted >= 0)
}

var isNil = func(t *testing.T, actual interface{}) {
	assert.Nil(t, actual)
}

var isString = func(t *testing.T, actual interface{}) {
	_, ok := actual.(string)
	assert.True(t, ok)
}

type eventFieldTest struct {
	key         string
	description string
	assertion   func(t *testing.T, actual interface{})
}

func testEventFields(t *testing.T, event beat.Event, eventFieldTests []eventFieldTest) {
	for _, eventFieldTest := range eventFieldTests {
		matcher := eventFieldTest
		name := fmt.Sprintf("%s %s", matcher.key, matcher.description)
		t.Run(name, func(t *testing.T) {
			actual, _ := event.Fields.GetValue(matcher.key) // ignore err to allow nil assert later
			matcher.assertion(t, actual)
		})
	}
}

func executeHTTPMonitorHostJob(t *testing.T, handlerFunc http.HandlerFunc, expectedStatus int, expectedBody string, expectedMonitorStatus string) {
	server := httptest.NewServer(handlerFunc)
	defer server.Close()

	config := common.NewConfig()
	config.SetString("urls", 0, server.URL)

	jobs, err := create(monitors.Info{}, config)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	job := jobs[0]

	event, jobRunners, err := job.Run()
	assert.Nil(t, err)
	assert.NotNil(t, jobRunners)

	parsedServerURL, err := url.Parse(server.URL)
	assert.Nil(t, err)
	serverPortInt, err := strconv.Atoi(parsedServerURL.Port())
	assert.Nil(t, err)
	serverPort := uint16(serverPortInt)

	fmt.Println(event.Fields)
	testEventFields(t, event, []eventFieldTest{
		{
			"tcp.port",
			"is the server port",
			exactlyEqual(serverPort),
		},
		{
			"tcp.rtt.connect.us",
			"isDuration",
			isDuration,
		},
		{
			"monitor.id",
			"is the server URL and actual proto",
			exactlyEqual("http@" + server.URL),
		},
		{
			"monitor.ip",
			"is a string", // Don't test the exact value, could shift with IPv6 only stack
			isString,
		},
		{
			"monitor.status",
			"is up",
			exactlyEqual(expectedMonitorStatus),
		},
		{
			"monitor.scheme",
			"is http",
			exactlyEqual("http"),
		},
		{
			"http.response.status_code",
			"is as expected",
			exactlyEqual(expectedStatus),
		},
		{
			"http.url",
			"is the server URL",
			exactlyEqual(server.URL),
		},
		{
			"http.rtt.content.us",
			"isDuration",
			isDuration,
		},
		{
			"http.rtt.response_header.us",
			"isDuration",
			isDuration,
		},
		{
			"http.rtt.total.us",
			"isDuration",
			isDuration,
		},
		{
			"http.rtt.validate.us",
			"isDuration",
			isDuration,
		},
		{
			"http.rtt.write_request.us",
			"isDuration",
			isDuration,
		},
	})
}

func TestOKJob(t *testing.T) {
	executeHTTPMonitorHostJob(t, helloWorldHandler, http.StatusOK, helloWorldBody, "up")
}

func TestBadGatewayJob(t *testing.T) {
	executeHTTPMonitorHostJob(t, badGatewayHandler, http.StatusBadGateway, badGatewayBody, "down")
}

func TestBadHostJob(t *testing.T) {
	config := common.NewConfig()
	ip := "192.0.2.0"
	url := "http://" + ip
	config.SetString("urls", 0, url)

	jobs, err := create(monitors.Info{}, config)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	job := jobs[0]

	event, jobRunners, err := job.Run()
	assert.Nil(t, err)
	assert.NotNil(t, jobRunners)

	fmt.Println(event.Fields)
	testEventFields(t, event, []eventFieldTest{
		{
			"error.message",
			"is a string",
			isString,
		},
		{
			"error.type",
			"is io",
			exactlyEqual("io"),
		},
		{
			"http.url",
			"is the url",
			exactlyEqual(url),
		},
		{
			"monitor.id",
			"is the server URL and actual proto",
			exactlyEqual("http@" + url),
		},
		{
			"monitor.ip",
			"is the exact ip",
			exactlyEqual(ip),
		},
		{
			"monitor.duration.us",
			"is a duration",
			isDuration,
		},
		{
			"monitor.status",
			"is down",
			exactlyEqual("down"),
		},
		{
			"monitor.scheme",
			"is http",
			exactlyEqual("http"),
		},
		{
			"http.rtt",
			"isDuration",
			isNil,
		},
		{
			"tcp.port",
			"is port 80",
			exactlyEqual(uint16(80)),
		},
	})
}
