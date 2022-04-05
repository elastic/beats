// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bay "github.com/elastic/bayeux"
	finput "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	serverURL string
)

func TestNewInputDone(t *testing.T) {
	config := common.MapStr{
		"channel_name":              "cometd-channel",
		"auth.oauth2.client.id":     "DEMOCLIENTID",
		"auth.oauth2.client.secret": "DEMOCLIENTSECRET",
		"auth.oauth2.user":          "salesforce_user",
		"auth.oauth2.password":      "P@$$w0₹D",
		"auth.oauth2.token_url":     "https://login.salesforce.com/services/oauth2/token",
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

	var msg bay.TriggerEvent
	msg.Data.Event.ReplayID = 1234
	msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	msg.Channel = "first-channel"

	config := map[string]interface{}{
		"channel_name":              "first-channel",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
	}

	r := http.HandlerFunc(oauth2Handler)
	server := httptest.NewServer(r)
	serverURL = server.URL
	config["auth.oauth2.token_url"] = server.URL + "/token"

	cfg := common.MustNewConfigFrom(config)

	input, err := NewInput(cfg, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	input.Run()
	for _, event := range []beat.Event{<-eventsCh} {
		require.NoError(t, err)
		assertEventMatches(t, msg, event)
	}
	server.Close()
}

func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = r.ParseForm()
	switch {
	case r.Method != "POST":
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
	case r.Method != "POST":
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
	case r.Method != "POST":
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
	case r.Method != "POST":
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
	if r.URL.Path == "/cometd/38.0" && string(body) == `{"channel": "/meta/handshake", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}` {
		oauth2ClientIdHandler(w, r)
		return
	} else if r.URL.Path == "/cometd/38.0" && string(body) == `{"channel": "/meta/connect", "connectionType": "long-polling", "clientId": "94b112sp7ph1c9s41mycpzik4rkj3"} ` {
		oauth2EventHandler(w, r)
		return
	} else if r.URL.Path == "/cometd/38.0" && string(body) == `{
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

func assertEventMatches(t *testing.T, expected bay.TriggerEvent, got beat.Event) {
	message, err := got.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(expected.Data.Payload), message)
}
