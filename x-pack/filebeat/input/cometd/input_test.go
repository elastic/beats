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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	firstChannel  = "channel_name1"
	secondChannel = "channel_name2"
)

var (
	serverURL string
	called1   uint64
	called2   uint64
	clientId  uint64
)

func TestInputDone(t *testing.T) {
	config := common.MapStr{
		"channel_name":              firstChannel,
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
		Fields: common.MapStr{
			"event": common.MapStr{
				"id":      "DEMOID",
				"created": time.Now().UTC(),
			},
			"message": "DEMOBODYFAIL",
		},
		Private: "DEMOBODYFAIL",
	}
	assert.NotEqual(t, event, makeEvent("DEMOID", "DEMOBODY"))
}

func TestSingleInput(t *testing.T) {
	defer atomic.StoreUint64(&called1, 0)
	defer atomic.StoreUint64(&called2, 0)
	defer atomic.StoreUint64(&clientId, 0)
	eventsCh := make(chan beat.Event)

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
	expected.Msg.Channel = firstChannel

	config := map[string]interface{}{
		"channel_name":              firstChannel,
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

	cfg := common.MustNewConfigFrom(config)

	input, err := NewInput(cfg, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()

	event := <-eventsCh
	assertEventMatches(t, expected, event)
	input.Stop()
}

func TestInputStop_Wait(t *testing.T) {
	defer atomic.StoreUint64(&called1, 0)
	defer atomic.StoreUint64(&called2, 0)
	defer atomic.StoreUint64(&clientId, 0)
	eventsCh := make(chan beat.Event)

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
	expected.Msg.Channel = firstChannel

	config := map[string]interface{}{
		"channel_name":              firstChannel,
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

	cfg := common.MustNewConfigFrom(config)

	input, err := NewInput(cfg, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	require.Equal(t, 0, bay.GetConnectedCount())
	input.Run()
	eventProcessing.Wait()
	require.Equal(t, 1, bay.GetConnectedCount())

	go func() {
		time.Sleep(100 * time.Millisecond) // let input.Stop() be executed.
		for range eventsCh {
		}
	}()

	input.Wait()
	require.Equal(t, 0, bay.GetConnectedCount())
}

func TestMultiInput(t *testing.T) {
	defer atomic.StoreUint64(&called1, 0)
	defer atomic.StoreUint64(&called2, 0)
	defer atomic.StoreUint64(&clientId, 0)
	eventsCh := make(chan beat.Event)

	const numMessages = 2

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

	var expected1 bay.MaybeMsg
	expected1.Msg.Data.Event.ReplayID = 1234
	expected1.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	expected1.Msg.Channel = firstChannel

	var expected2 bay.MaybeMsg
	expected2.Msg.Data.Event.ReplayID = 1234
	expected2.Msg.Data.Payload = []byte(`{"CountryIso": "US"}`)
	expected2.Msg.Channel = secondChannel

	config1 := map[string]interface{}{
		"channel_name":              firstChannel,
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}
	config2 := map[string]interface{}{
		"channel_name":              secondChannel,
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
	config1["auth.oauth2.token_url"] = serverURL + "/token"
	config2["auth.oauth2.token_url"] = serverURL + "/token"

	// get common config
	cfg1 := common.MustNewConfigFrom(config1)
	cfg2 := common.MustNewConfigFrom(config2)

	var inputContext finput.Context

	// initialize inputs
	input1, err := NewInput(cfg1, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input1)

	input2, err := NewInput(cfg2, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input2)

	require.Equal(t, 0, bay.GetConnectedCount())
	// run input
	input1.Run()

	// run input
	input2.Run()
	eventProcessing.Wait()
	require.Equal(t, 2, bay.GetConnectedCount())

	go func() {
		time.Sleep(4 * time.Second)
		event := <-eventsCh
		assertEventMatches(t, expected1, event)
	}()

	go func() {
		time.Sleep(5 * time.Second)
		event := <-eventsCh
		assertEventMatches(t, expected2, event)
	}()

	input1.Wait()
	input2.Wait()

	require.Equal(t, 0, bay.GetConnectedCount())
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
	default:
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

func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		response := `{"instance_url": "` + serverURL + `", "expires_in": "60", "access_token": "abcd"}`
		_, _ = w.Write([]byte(response))
	}
}

func oauth2ClientIdHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		switch clientId {
		case 0:
			atomic.StoreUint64(&clientId, 1)
			_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id1","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
		case 1:
			atomic.StoreUint64(&clientId, 0)
			_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"client_id2","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
		default:
		}
	}
}

func oauth2SubscribeHandlerChannel1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		_, _ = w.Write([]byte(`[{"clientId": "client_id1", "channel": "/meta/subscribe", "subscription": "channel_name1", "successful":true}]`))
	}
}

func oauth2SubscribeHandlerChannel2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		_, _ = w.Write([]byte(`[{"clientId": "client_id2", "channel": "/meta/subscribe", "subscription": "channel_name2", "successful":true}]`))
	}
}

func oauth2EventHandlerChannel1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		if called1 < 1 {
			atomic.AddUint64(&called1, 1)
			_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "channel_name1"}]`))
		}
	}
}

func oauth2EventHandlerChannel2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		if called2 < 1 {
			atomic.AddUint64(&called2, 1)
			_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "US"}, "event": {"replayId":1234}}, "channel": "channel_name2"}]`))
		}
	}
}

func oauth2Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/token" {
		oauth2TokenHandler(w, r)
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == `{"channel": "/meta/handshake", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}` {
		oauth2ClientIdHandler(w, r)
		return
	} else if string(body) == `{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "client_id1"} ` {
		oauth2EventHandlerChannel1(w, r)
		return
	} else if string(body) == `{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "client_id2"} ` {
		oauth2EventHandlerChannel2(w, r)
		return
	} else if string(body) == `{
								"channel": "/meta/subscribe",
								"subscription": "channel_name1",
								"clientId": "client_id1",
								"ext": {
									"replay": {"channel_name1": "-1"}
									}
								}` {
		oauth2SubscribeHandlerChannel1(w, r)
		return
	} else if string(body) == `{
								"channel": "/meta/subscribe",
								"subscription": "channel_name2",
								"clientId": "client_id2",
								"ext": {
									"replay": {"channel_name2": "-1"}
									}
								}` {
		oauth2SubscribeHandlerChannel2(w, r)
		return
	}
}

func assertEventMatches(t *testing.T, expected bay.MaybeMsg, got beat.Event) {
	message, err := got.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.Msg.Data.Payload), message)
}
