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
)

func TestNewInputDone(t *testing.T) {
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

func TestNewInput_Run(t *testing.T) {
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
	defer input.Stop()
	event := <-eventsCh
	message, err := event.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.Msg.Data.Payload), message)
}

func TestNewMultiInput_Run(t *testing.T) {
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
	expected.Msg.Channel = firstChannel

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
	serverURL = server.URL
	config1["auth.oauth2.token_url"] = serverURL + "/token"
	config2["auth.oauth2.token_url"] = serverURL + "/token"

	// get common config
	cfg1 := common.MustNewConfigFrom(config1)
	cfg2 := common.MustNewConfigFrom(config2)

	var inputContext finput.Context

	// intialize inputs
	input1, err := NewInput(cfg1, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input1)

	input2, err := NewInput(cfg2, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input2)

	// run input
	input1.Run()
	defer input1.Stop()

	event1 := <-eventsCh
	assertEventMatches(t, expected, event1)

	// run input
	input2.Run()
	defer input2.Stop()

	event2 := <-eventsCh
	assertEventMatches(t, expected, event2)

	// close server
	server.Close()
}

// TestNewInput_Run_Wait to test input wait
func TestNewInput_Run_Wait(t *testing.T) {
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

	var msg bay.MaybeMsg
	msg.Msg.Data.Event.ReplayID = 1234
	msg.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	msg.Msg.Channel = firstChannel

	config := map[string]interface{}{
		"channel_name":              firstChannel,
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	r := http.HandlerFunc(oauth2Handler)
	server := httptest.NewServer(r)
	serverURL = server.URL
	config["auth.oauth2.token_url"] = serverURL + "/token"

	cfg := common.MustNewConfigFrom(config)

	input, err := NewInput(cfg, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	// run input
	input.Run()

	go func() {
		time.Sleep(100 * time.Millisecond) // let input.Stop() be executed.
		input.Wait()
	}()

	for range []beat.Event{<-eventsCh} {
	}
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
		time.Sleep(1000 * time.Millisecond)
		input.Wait()
	}()

	time.Sleep(1000 * time.Millisecond) // let input.Stop() be executed.
	select {
	case <-workerCtx.Done():
	default:
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
		_, _ = w.Write([]byte(`[{"ext":{"replay":true,"payload.format":true},"minimumVersion":"1.0","clientId":"94b112sp7ph1c9s41mycpzik4rkj3","supportedConnectionTypes":["long-polling"],"channel":"/meta/handshake","version":"1.0","successful":true}]`))
	}
}

func oauth2SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		_, _ = w.Write([]byte(`[{"clientId": "94b112sp7ph1c9s41mycpzik4rkj3", "channel": "/meta/subscribe", "subscription": "/event/LoginEventStream", "successful":true}]`))
	}
}

func oauth2EventHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"wrong method"}`))
	default:
		_, _ = w.Write([]byte(`[{"data": {"payload": {"CountryIso": "IN"}, "event": {"replayId":1234}}, "channel": "/event/LoginEventStream"}]`))
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
	} else if string(body) == `{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "94b112sp7ph1c9s41mycpzik4rkj3"} ` {
		oauth2EventHandler(w, r)
		return
	} else if string(body) == `{
								"channel": "/meta/subscribe",
								"subscription": "first-channel",
								"clientId": "94b112sp7ph1c9s41mycpzik4rkj3",
								"ext": {
									"replay": {"first-channel": "-1"}
									}
								}` {
		oauth2SubscribeHandler(w, r)
		return
	}
}

func assertEventMatches(t *testing.T, expected bay.MaybeMsg, got beat.Event) {
	message, err := got.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.Msg.Data.Payload), message)
}
