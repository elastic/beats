// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// A lot of code in this file would be updated as a part of data collection
// mechanism implementation.
// Please do not review it, as the current code would test the authentication
// mechanism only.

package cometd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	inputName = "cometd"
)

type cometdInput struct {
	config

	log      *logp.Logger
	inputCtx context.Context // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on pubsub worker goroutine.

	ackedCount *atomic.Uint32                   // Total number of successfully ACKed pubsub messages.
	Transport  httpcommon.HTTPTransportSettings `config:",inline"`
	Retry      retryConfig                      `config:"retry"`
}

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

// NewInput creates a new CometD Pub/Sub input that consumes events from
// a topic subscription.
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err = cfg.Unpack(&conf); err != nil {
		return nil, fmt.Errorf("cometd config: unpacking of config failed: %v", err)
	}

	logger := logp.NewLogger("cometd").With(
		"pubsub_channel", conf.ChannelName)

	// Wrap input.Context's Done channel with a context.Context. This goroutine
	// stops with the parent closes the Done channel.
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
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
		ackedCount:   atomic.NewUint32(0),
	}

	in.log.Info("Initialized cometD input.")
	return in, nil
}

// Run starts the pubsub input worker then returns. Only the first invocation
// will ever start the pubsub worker.
func (in *cometdInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("Pub/Sub input worker has started.")
			defer in.log.Info("Pub/Sub input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

// Stop stops the pubsub input and waits for it to fully stop.
func (in *cometdInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *cometdInput) Wait() {
	in.Stop()
}

func (in *cometdInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	client, err := in.newPubsubClient(ctx)
	if err != nil {
		return fmt.Errorf("cometd client: error creating pub-sub client: %v", err)
	}

	in.log.Debug("client successfully created")

	// For testing http client
	// Start of the test code for authentication and client creation
	baseUrl := "<put base URL here>"
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		in.log.Errorf("Error on request generation: %v", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		in.log.Errorf("Error on response: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		in.log.Errorf("Error while reading the response bytes: %v", err)
	}

	in.log.Infof("Response body: %v", string(body))
	// End of the test code for authentication and client creation
	return nil
}

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

func (in *cometdInput) newPubsubClient(ctx context.Context) (*http.Client, error) {
	// Make retryable HTTP client
	netHTTPClient, err := in.Transport.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithKeepaliveSettings{Disable: true},
	)
	if err != nil {
		return nil, fmt.Errorf("cometd client: error on newHTTPClient: %v", err)
	}

	client := &retryablehttp.Client{
		HTTPClient: netHTTPClient,
		CheckRetry: retryablehttp.DefaultRetryPolicy,
		Backoff:    retryablehttp.DefaultBackoff,
	}

	authClient, err := in.config.Auth.OAuth2.client(ctx, client.StandardClient())
	if err != nil {
		return nil, fmt.Errorf("cometd client: error on authClient: client not created: %v", err)
	}
	return authClient, nil
}
