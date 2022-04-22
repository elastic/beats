// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	bay "github.com/elastic/bayeux"
)

const (
	inputName = "cometd"
)

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *cometdInput) Run() {
	var err error
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("Input worker has started.")
			defer in.log.Info("Input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			in.b = bay.Bayeux{}
			in.creds, err = bay.GetSalesforceCredentials(in.authParams)
			if err != nil {
				in.log.Error("not able to get access token: ", err)
				return
			}
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

func (in *cometdInput) run() error {
	in.msgCh = in.b.Channel(in.workerCtx, in.msgCh, "-1", *in.creds, in.config.ChannelName)
	for e := range in.msgCh {
		if e.Failed() {
			return fmt.Errorf("error collecting events: %w", e.Err)
		}
		if !e.Msg.Successful {
			var event Event
			// To handle the last response where the object received was empty
			if e.Msg.Data.Payload == nil {
				return nil
			}

			// Convert json.RawMessage response to []byte
			msg, err := e.Msg.Data.Payload.MarshalJSON()
			if err != nil {
				return fmt.Errorf("JSON error: %w", err)
			}

			// Extract event IDs from json.RawMessage
			err = json.Unmarshal(e.Msg.Data.Payload, &event)
			if err != nil {
				return fmt.Errorf("error while parsing JSON: %w", err)
			}
			if ok := in.outlet.OnEvent(makeEvent(event.EventId, e.Msg.Channel, string(msg))); !ok {
				in.log.Debug("OnEvent returned false. Stopping input worker.")
				return fmt.Errorf("error ingesting data to elasticsearch")
			}
		}
	}
	return nil
}

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(fmt.Errorf("failed to register %v input: %w", inputName, err))
	}
}

// NewInput creates a new CometD input that consumes events from
// a topic subscription.
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	var authParams bay.AuthenticationParameters
	authParams.ClientID = conf.Auth.OAuth2.ClientID
	authParams.ClientSecret = conf.Auth.OAuth2.ClientSecret
	authParams.Username = conf.Auth.OAuth2.User
	authParams.Password = conf.Auth.OAuth2.Password
	authParams.TokenURL = conf.Auth.OAuth2.TokenURL

	logger := logp.NewLogger("cometd").With(
		"pubsub_channel", conf.ChannelName)

	// Wrap input.Context's Done channel with a context.Context. This goroutine
	// stops with the parent closes the Done channel.
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		<-inputContext.Done
		cancelInputCtx()
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	in := &cometdInput{
		config:       conf,
		log:          logger,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		authParams:   authParams,
	}

	// Creating a new channel for cometd input.
	in.msgCh = make(chan bay.MaybeMsg, 1)

	// Build outlet for events.
	in.outlet, err = connector.Connect(cfg)
	if err != nil {
		return nil, err
	}
	in.log.Infof("Initialized %s input.", inputName)
	return in, nil
}

// Stop stops the input and waits for it to fully stop.
func (in *cometdInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *cometdInput) Wait() {
	in.Stop()
}

type cometdInput struct {
	config

	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on worker goroutine.

	msgCh      chan bay.MaybeMsg
	b          bay.Bayeux
	creds      *bay.Credentials
	authParams bay.AuthenticationParameters
}

type Event struct {
	EventId string `json:"EventIdentifier"`
}

func makeEvent(id string, channel string, body string) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"event": common.MapStr{
				"id":      id,
				"created": time.Now().UTC(),
			},
			"message": body,
			"cometd": common.MapStr{
				"channel_name": channel,
			},
		},
		Private: body,
	}
	event.SetID(id)

	return event
}
