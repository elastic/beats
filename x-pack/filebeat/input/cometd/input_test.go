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

	input, err := NewInput(cfg, connector, inputContext)
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

	input, err := NewInput(cfg, connector, inputContext)
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

func TestMultiInput(t *testing.T) {
	expectedHTTPEventCount = 2
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

	var expected1 bay.MaybeMsg
	expected1.Msg.Data.Event.ReplayID = 1234
	expected1.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected1.Msg.Channel = "channel_name"

	config := map[string]interface{}{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	// create Server
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
			if called < uint64(expectedHTTPEventCount) {
				if called == 0 {
					atomic.AddUint64(&called, 1)
					_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`))
					return
				} else if called == 1 {
					atomic.AddUint64(&called, 1)
					_, _ = w.Write([]byte(`[{"data": {"sobject": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`))
					return
				}
			}
			_, _ = w.Write([]byte(`{}`))
			return
		case "/meta/subscribe":
			if called == 0 {
				_, _ = w.Write([]byte(`[{"clientId": "client_id", "channel": "/meta/subscribe", "subscription": "channel_name", "successful":true}]`))
			} else if called == 1 {
				_, _ = w.Write([]byte(`[{"clientId": "client_id", "channel": "/meta/subscribe", "subscription": "channel_name1", "successful":true}]`))
			}
			return
		default:
		}
	})
	server := httptest.NewServer(r)
	defer server.Close()
	serverURL = server.URL
	config["auth.oauth2.token_url"] = serverURL + "/token"

	// get common config
	cfg1 := conf.MustNewConfigFrom(config)
	config["channel_name"] = "channel_name1"
	cfg2 := conf.MustNewConfigFrom(config)

	var inputContext finput.Context

	// initialize inputs
	input1, err := NewInput(cfg1, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input1)

	input2, err := NewInput(cfg2, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input2)

	require.Equal(t, 0, bay.GetConnectedCount())

	got := 0
	go func() {
		// run input
		input1.Run()
		defer input1.Stop()

		// run input
		input2.Run()
		defer input2.Stop()

		for _, event := range []beat.Event{<-eventsCh, <-eventsCh} {
			message, err := event.GetValue("message")
			require.NoError(t, err)
			require.Equal(t, string(expected1.Msg.Data.Payload), message)
			got++
		}
	}()
	time.Sleep(time.Second)
	if got < 2 {
		require.NoError(t, fmt.Errorf("not able to get events."))
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
		if called < uint64(expectedHTTPEventCount) {
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

	input, err := NewInput(cfg, connector, inputContext)
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

	input, err := NewInput(cfg, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()
	go func() {
		<-eventsCh
		assert.Error(t, fmt.Errorf("there should be no events"))
	}()

	// wait for run to return error or event
	time.Sleep(100 * time.Millisecond)

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
