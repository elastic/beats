// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	r := http.HandlerFunc(oauth2Handler)
	server := httptest.NewServer(r)
	defer server.Close()
	serverURL = server.URL
	config["auth.oauth2.token_url"] = serverURL + "/token"

	// get common config
	cfg1 := conf.MustNewConfigFrom(config)
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
			assertEventMatches(t, expected1, event)
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
	if r.URL.Path == "/token" {
		response := `{"instance_url": "` + serverURL + `", "expires_in": "60", "access_token": "abcd"}`
		_, _ = w.Write([]byte(response))
		return
	}
	body, _ := ioutil.ReadAll(r.Body)

	switch string(body) {
	case `{"channel": "/meta/handshake", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}`:
		_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
		return
	case `{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "client_id"} `:
		if called < uint64(expectedHTTPEventCount) {
			atomic.AddUint64(&called, 1)
			_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name"}]`))
		} else {
			_, _ = w.Write([]byte(`{}`))
		}
		return
	case `{
								"channel": "/meta/subscribe",
								"subscription": "channel_name",
								"clientId": "client_id",
								"ext": {
									"replay": {"channel_name": "-1"}
									}
								}`:
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
