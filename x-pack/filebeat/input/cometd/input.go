// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	bay "github.com/elastic/bayeux"
	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	inputName = "cometd"

	// retryInterval is the minimum duration between pub/sub client retries.
	retryInterval = 30 * time.Second
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

			rt := rate.NewLimiter(rate.Every(retryInterval), 1)

			for in.workerCtx.Err() == nil {
				// Rate limit.
				if err := rt.Wait(in.workerCtx); err != nil {
					continue
				}

				// Creating a new channel for cometd input.
				in.msgCh = make(chan bay.MaybeMsg, 1)

				in.creds, err = bay.GetSalesforceCredentials(in.authParams)
				if err != nil {
					in.log.Errorw("not able to get access token", "error", err)
					continue
				}

				if err := in.run(); err != nil {
					if in.workerCtx.Err() == nil {
						in.log.Errorw("Restarting failed CometD input worker.", "error", err)
						continue
					}

					// Log any non-cancellation error before stopping.
					if !errors.Is(err, context.Canceled) {
						in.log.Errorw("got error while running input", "error", err)
					}
				}
			}
		}()
	})
}

func (in *cometdInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()
	// Ticker with 5 seconds to avoid log too many warnings
	ticker := time.NewTicker(5 * time.Second)
	in.msgCh = in.b.Channel(ctx, in.msgCh, "-1", *in.creds, in.config.ChannelName)
	for e := range in.msgCh {
		if e.Failed() {
			// if err bayeux library returns recoverable error, do not close input.
			// instead continue with connection warning
			if !strings.Contains(e.Error(), "trying again") {
				return fmt.Errorf("error collecting events: %w", e.Err)
			}
			// log warning every 5 seconds only to avoid to many unnecessary logs
			select {
			case <-ticker.C:
				in.log.Errorw("Retrying...! facing issue while collecting data from CometD", "error", e.Error())
			default:
			}
		} else if !e.Msg.Successful {
			var event event
			var msg []byte
			var err error
			// Convert json.RawMessage response to []byte
			if e.Msg.Data.Payload != nil {
				msg, err = e.Msg.Data.Payload.MarshalJSON()
				if err != nil {
					in.log.Errorw("invalid JSON", "error", err)
					continue
				}
			} else if e.Msg.Data.Object != nil {
				msg, err = e.Msg.Data.Object.MarshalJSON()
				if err != nil {
					in.log.Errorw("invalid JSON", "error", err)
					continue
				}
			} else {
				// To handle the last response where the object received was empty
				return nil
			}

			// Extract event IDs from json.RawMessage
			err = json.Unmarshal(msg, &event)
			if err != nil {
				in.log.Errorw("error while parsing JSON", "error", err)
				continue
			}
			if ok := in.outlet.OnEvent(makeEvent(event.EventId, e.Msg.Channel, string(msg))); !ok {
				in.log.Debug("OnEvent returned false. Stopping input worker.")
				cancel()
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
	cfg *conf.C,
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

	logger := logp.NewLogger(inputName).With(
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

type event struct {
	EventId string `json:"EventIdentifier"`
}

func makeEvent(id string, channel string, body string) beat.Event {
	e := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: mapstr.M{
			"event": mapstr.M{
				"id":      id,
				"created": time.Now().UTC(),
			},
			"message": body,
			"cometd": mapstr.M{
				"channel_name": channel,
			},
		},
		Private: body,
	}
	e.SetID(id)

	return e
}
