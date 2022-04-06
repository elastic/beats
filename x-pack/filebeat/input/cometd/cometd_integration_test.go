// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"fmt"
	"sync"
	"testing"

	bay "github.com/elastic/bayeux"
	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/stretchr/testify/require"
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
	logp.TestingSetup(logp.WithSelectors("cometd input", "cometd"))

	// Setup the input config.
	config := common.MustNewConfigFrom(common.MapStr{
		"channel_name":              "channel_name",
		"auth.oauth2.client.id":     "client.id",
		"auth.oauth2.client.secret": "client.secret",
		"auth.oauth2.user":          "user",
		"auth.oauth2.password":      "password",
		"auth.oauth2.token_url":     "http://127.0.0.1:8080/token",
	})

	// Route input events through our captor instead of sending through ES.
	eventsCh := make(chan beat.Event)
	defer close(eventsCh)

	captor := newEventCaptor(eventsCh)
	defer captor.Close()

	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
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

	// Run the input.
	input.Run()

	verifiedCh := make(chan struct{})
	defer close(verifiedCh)

	var msg bay.TriggerEvent
	msg.Data.Event.ReplayID = 1234
	msg.Data.Payload = []byte(`{"CountryIso": "IN"}`)
	msg.Channel = "first-channel"

	for _, event := range []beat.Event{<-eventsCh} {
		require.NoError(t, err)
		message, err := event.GetValue("message")
		require.NoError(t, err)
		require.Equal(t, string(msg.Data.Payload), message)
	}
}
