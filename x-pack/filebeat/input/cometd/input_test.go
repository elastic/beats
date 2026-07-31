// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bay "github.com/elastic/bayeux"
	finput "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	serverURL              string
	expectedHTTPEventCount int
	called                 uint64
)

func TestInputDone(t *testing.T) {
	config := mapstr.M{
		"channel_name":              "channel_channel",
		"auth.oauth2.client.id":     "DEMOCLIENTID",
		"auth.oauth2.client.secret": "DEMOCLIENTSECRET",
		"auth.oauth2.user":          "salesforce_user",
		"auth.oauth2.password":      "pwd",
		"auth.oauth2.token_url":     "https://example.com/token",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}

func TestMakeEventFailure(t *testing.T) {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: mapstr.M{
			"event": mapstr.M{
				"id":      "DEMOID",
				"created": time.Now().UTC(),
			},
			"message": "DEMOBODYFAIL",
			"cometd": mapstr.M{
				"channel_name": "DEMOCHANNEL",
			},
		},
		Private: "DEMOBODYFAIL",
	}
	assert.NotEqual(t, event, makeEvent("DEMOCHANNEL", "DEMOID", "DEMOBODY"))
}

const payloadFormatEnvVar = "BEATS_COMETD_PAYLOAD_FORMAT"

// connectResponses maps a Salesforce payload format to a /meta/connect response using it.
var connectResponses = map[string]string{
	"payload": `[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`,
	"sobject": `[{"data": {"sobject": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`,
}

func TestInputPayloadFormats(t *testing.T) {
	// github.com/elastic/bayeux keeps subscription state in unsynchronized package globals
	// and its connect goroutine outlives Stop(), so two clients in one test binary race
	// under -race no matter how the tests are ordered.
	for format := range connectResponses {
		t.Run(format, func(t *testing.T) {
			//nolint:gosec // subprocess re-exec of this test binary with fixed args
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestInputPayloadFormatsProcess$", "-test.count=1")
			cmd.Env = append(os.Environ(), payloadFormatEnvVar+"="+format)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "payload format %q subprocess failed:\n%s", format, out)
		})
	}
}

func TestInputPayloadFormatsProcess(t *testing.T) {
	format := os.Getenv(payloadFormatEnvVar)
	if format == "" {
		t.Skip("subprocess helper for TestInputPayloadFormats")
	}
	connectResponse, ok := connectResponses[format]
	require.True(t, ok, "unknown payload format %q", format)
	runInputPayloadFormatCase(t, connectResponse)
}

func runInputPayloadFormatCase(t *testing.T, connectResponse string) {
	t.Helper()

	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}

	var expected bay.MaybeMsg
	expected.Msg.Data.Event.ReplayID = 1234
	expected.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected.Msg.Channel = "channel_name"

	config := map[string]any{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	// The server serves each request on its own goroutine.
	var connectEventSent atomic.Bool

	r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_ = r.ParseForm()
		if getTokenHandler(w, r) {
			return
		}
		body, _ := io.ReadAll(r.Body)
		data := getBayData(body)

		switch data.Channel {
		case "/meta/handshake":
			_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
			return
		case "/meta/connect":
			if connectEventSent.Swap(true) {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(connectResponse))
			return
		case "/meta/subscribe":
			_, _ = w.Write([]byte(`[{"clientId": "client_id", "channel": "/meta/subscribe", "subscription": "channel_name", "successful":true}]`))
			return
		default:
		}
	})
	server := httptest.NewServer(r)
	defer server.Close()
	serverURL = server.URL
	config["auth.oauth2.token_url"] = serverURL + "/token"

	cfg := conf.MustNewConfigFrom(config)

	var inputContext finput.Context

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(cfg, connector, inputContext, logger)
	require.NoError(t, err, "NewInput for payload format case")
	require.NotNil(t, input, "input for payload format case")

	input.Run()
	assertEventMatches(t, expected, <-eventsCh)
	input.Stop()
}

func TestSingleInput(t *testing.T) {
	expectedHTTPEventCount = 1
	defer atomic.StoreUint64(&called, 0)
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	var expected bay.MaybeMsg
	expected.Msg.Data.Event.ReplayID = 1234
	expected.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected.Msg.Channel = "channel_name"

	config := map[string]interface{}{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	r := http.HandlerFunc(oauth2Handler)
	server := httptest.NewServer(r)
	defer server.Close()

	serverURL = server.URL
	config["auth.oauth2.token_url"] = server.URL + "/token"

	cfg := conf.MustNewConfigFrom(config)

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(cfg, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	event := <-eventsCh
	assertEventMatches(t, expected, event)
	input.Stop()
}

func TestInputStop_Wait(t *testing.T) {
	t.Skip("Flaky test https://github.com/elastic/beats/issues/37987")

	expectedHTTPEventCount = 1
	defer atomic.StoreUint64(&called, 0)
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	const numMessages = 1

	var eventProcessing sync.WaitGroup
	eventProcessing.Add(numMessages)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventProcessing.Done()
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	var expected bay.MaybeMsg
	expected.Msg.Data.Event.ReplayID = 1234
	expected.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected.Msg.Channel = "channel_name"

	config := map[string]interface{}{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	r := http.HandlerFunc(oauth2Handler)
	server := httptest.NewServer(r)
	defer server.Close()

	serverURL = server.URL
	config["auth.oauth2.token_url"] = server.URL + "/token"

	cfg := conf.MustNewConfigFrom(config)

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(cfg, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	require.Equal(t, 0, bay.GetConnectedCount())
	input.Run()
	eventProcessing.Wait()
	require.Equal(t, 1, bay.GetConnectedCount())

	var waitForEventCollection sync.WaitGroup
	var waitForConnections sync.WaitGroup
	waitForEventCollection.Add(1)
	waitForConnections.Add(1)
	go func() {
		require.Equal(t, 1, bay.GetConnectedCount()) // current open channels count should be 1
		event := <-eventsCh
		assertEventMatches(t, expected, event) // wait for single event
		waitForEventCollection.Done()
		time.Sleep(100 * time.Millisecond)           // let input.Stop() be executed.
		require.Equal(t, 0, bay.GetConnectedCount()) // current open channels count should be 0
		waitForConnections.Done()
	}()

	waitForEventCollection.Wait()
	input.Wait()
	waitForConnections.Wait()
}

func TestStop(t *testing.T) {
	conf := defaultConfig()
	logger := logp.NewLogger("test")
	authParams := bay.AuthenticationParameters{}
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	workerCtx, workerCancel := context.WithCancel(inputCtx)
	defer cancelInputCtx()
	input := &cometdInput{
		config:       conf,
		log:          logger,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		authParams:   authParams,
	}
	input.msgCh = make(chan bay.MaybeMsg)

	input.Stop()
	select {
	case <-workerCtx.Done():
	case <-time.After(time.Second): // let input.Stop() be executed.
		require.NoError(t, fmt.Errorf("input is not stopped."))
	}
}

func TestWait(t *testing.T) {
	conf := defaultConfig()
	logger := logp.NewLogger("test")
	authParams := bay.AuthenticationParameters{}
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	workerCtx, workerCancel := context.WithCancel(inputCtx)
	defer cancelInputCtx()

	input := &cometdInput{
		config:       conf,
		log:          logger,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		authParams:   authParams,
	}
	input.msgCh = make(chan bay.MaybeMsg)

	go func() {
		input.Wait()
	}()

	select {
	case <-workerCtx.Done():
	case <-time.After(time.Second): // let input.Stop() be executed.
		require.NoError(t, fmt.Errorf("input is not stopped."))
	}
}

func oauth2Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	if getTokenHandler(w, r) {
		return
	}
	body, _ := io.ReadAll(r.Body)
	data := getBayData(body)

	switch data.Channel {
	case "/meta/handshake":
		_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
		return
	case "/meta/connect":
		if called < uint64(expectedHTTPEventCount) { //nolint:gosec //Safe to ignore in tests
			atomic.AddUint64(&called, 1)
			_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
		return
	case "/meta/subscribe":
		_, _ = w.Write([]byte(`[{"clientId": "client_id", "channel": "/meta/subscribe", "subscription": "channel_name", "successful":true}]`))
		return
	default:
	}
}

func assertEventMatches(t *testing.T, expected bay.MaybeMsg, got beat.Event) {
	message, err := got.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.Msg.Data.Payload), message)
}

func TestMultiEventForEOFRetryHandlerInput(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/34956")
	var err error

	expectedEventCount := 2

	eventsCh := make(chan beat.Event, expectedEventCount)
	signal := make(chan struct{}, 1)
	defer close(signal)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventsCh <- event
			signal <- struct{}{}
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	var expected bay.MaybeMsg
	expected.Msg.Data.Event.ReplayID = 1234
	expected.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected.Msg.Channel = "channel_name"

	config := map[string]interface{}{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	i := 0
	var server *httptest.Server
	r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_ = r.ParseForm()
		if getTokenHandler(w, r) {
			return
		}
		body, _ := io.ReadAll(r.Body)
		data := getBayData(body)

		switch data.Channel {
		case "/meta/handshake":
			_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
			return
		case "/meta/connect":
			if i == 0 {
				_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`))
				i++
				return
			}
			_, _ = w.Write([]byte(`{}`))
			return
		case "/meta/subscribe":
			_, _ = w.Write([]byte(`[{"clientId": "client_id", "channel": "/meta/subscribe", "subscription": "channel_name", "successful":true}]`))
			return
		default:
		}
	})

	server, err = newTestServer("", r)
	assert.NoError(t, err)
	serverURL = server.URL

	config["auth.oauth2.token_url"] = server.URL + "/token"

	cfg := conf.MustNewConfigFrom(config)

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(cfg, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	// close previous connection
	<-signal
	server.CloseClientConnections()
	server.Close()
	time.Sleep(100 * time.Millisecond)

	// restart connection for new events
	i = 0
	server, err = newTestServer(strings.Split(serverURL, "http://")[1], r)
	for err != nil {
		server, err = newTestServer(strings.Split(serverURL, "http://")[1], r)
	}
	<-signal
	server.CloseClientConnections()
	server.Close()

	close(eventsCh)

	go func() {
		for j := 0; j < expectedEventCount; j++ {
			event := <-eventsCh
			assertEventMatches(t, expected, event)
		}
		signal <- struct{}{}
	}()
	<-signal
	input.Stop()
}

func newTestServer(URL string, handler http.Handler) (*httptest.Server, error) {
	server := httptest.NewUnstartedServer(handler)
	if URL != "" {
		l, err := net.Listen("tcp", URL)
		if err != nil {
			return nil, err
		}
		server.Listener.Close()
		server.Listener = l
	}
	server.Start()
	return server, nil
}

func TestNegativeCases(t *testing.T) {
	expectedHTTPEventCount = 1
	defer atomic.StoreUint64(&called, 0)
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	outlet := &mockedOutleter{
		onEventHandler: func(event beat.Event) bool {
			eventsCh <- event
			return true
		},
	}
	connector := &mockedConnector{
		outlet: outlet,
	}
	var inputContext finput.Context

	var expected bay.MaybeMsg
	expected.Msg.Data.Event.ReplayID = 1234
	expected.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected.Msg.Channel = "channel_name"

	config := map[string]interface{}{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_ = r.ParseForm()
		if getTokenHandler(w, r) {
			return
		}
		body, _ := io.ReadAll(r.Body)
		data := getBayData(body)

		switch data.Channel {
		case "/meta/handshake":
			_, _ = w.Write([]byte(`{}`))
			return
		default:
		}
	})
	server := httptest.NewServer(r)
	defer server.Close()

	serverURL = server.URL
	config["auth.oauth2.token_url"] = server.URL + "/token"

	cfg := conf.MustNewConfigFrom(config)

	logger := logptest.NewTestingLogger(t, "")
	input, err := NewInput(cfg, connector, inputContext, logger)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	// wait for run to return error or event
	select {
	case <-eventsCh:
		t.Error("expected no events, but received one")
	case <-time.After(100 * time.Millisecond):
		// Expected: no events received.
	}

	input.Stop()
}

func getTokenHandler(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/token" {
		response := `{"instance_url": "` + serverURL + `", "expires_in": "60", "access_token": "abcd"}`
		_, _ = w.Write([]byte(response))
		return true
	}
	return false
}

func getBayData(body []byte) *bay.Subscription {
	var data bay.Subscription
	err := json.Unmarshal(body, &data)
	if err != nil {
		return nil
	}

	return &data
}
