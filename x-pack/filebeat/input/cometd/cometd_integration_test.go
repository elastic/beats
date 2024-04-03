// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package cometd

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"

	bay "github.com/elastic/bayeux"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type eventCaptor struct {
	c         chan struct{}
	closeOnce sync.Once
	closed    bool
	events    chan beat.Event
}

func newEventCaptor(events chan beat.Event) channel.Outleter {
	return &eventCaptor{
		c:      make(chan struct{}),
		events: events,
	}
}

func (ec *eventCaptor) OnEvent(event beat.Event) bool {
	ec.events <- event
	return true
}

func (ec *eventCaptor) Close() error {
	ec.closeOnce.Do(func() {
		ec.closed = true
		close(ec.c)
	})
	return nil
}

func (ec *eventCaptor) Done() <-chan struct{} {
	return ec.c
}

func TestInput(t *testing.T) {
	t.Skip("flaky test: https://github.com/elastic/beats/issues/33423")
	logp.TestingSetup(logp.WithSelectors("cometd input", "cometd"))

	// Setup the input config.
	config := conf.MustNewConfigFrom(mapstr.M{
		"channel_name":              "channel_name1",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
		"auth.oauth2.token_url":     "http://localhost:8080/token",
	})

	// Route input events through our captor instead of sending through ES.
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	captor := newEventCaptor(eventsCh)
	defer captor.Close()

	connector := channel.ConnectorFunc(func(_ *conf.C, _ beat.ClientConfig) (channel.Outleter, error) {
		return channel.SubOutlet(captor), nil
	})

	// Mock the context.
	inputContext := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	// Setup the input
	input, err := NewInput(config, connector, inputContext)
	require.NoError(t, err)
	require.NotNil(t, input)

	var msg bay.MaybeMsg
	msg.Msg.Data.Event.ReplayID = 1234
	msg.Msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	msg.Msg.Channel = "channel_name1"

	// Run the input.
	input.Run()

	event := <-eventsCh

	val, err := event.GetValue("message")
	require.NoError(t, err)
	require.Equal(t, string(msg.Msg.Data.Payload), val)

	input.Stop()
}
