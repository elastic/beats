// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package misp

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	inputName = "misp"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

type mispInput struct {
	config

	log      *logp.Logger
	outlet   channel.Outleter // Output of received misp messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on misp worker goroutine.

	msgCount *atomic.Uint32 // Total number of received MISP messages.
}

// NewInput creates a new Google Cloud Pub/Sub input that consumes events from
// a topic subscription.
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	// Build outlet for events.
	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

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

	in := &mispInput{
		config: conf,
		log: logp.NewLogger("misp").With(
			"misp_url", conf.Url),
		outlet:       out,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		msgCount:     atomic.NewUint32(0),
	}

	in.log.Info("Initialized misp input.")
	return in, nil
}

// Run starts the misp input worker then returns. Only the first invocation
// will ever start the misp worker.
func (in *mispInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("misp input worker has started.")
			defer in.log.Info("misp input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

func (in *mispInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	req, _ := http.NewRequest(http.MethodGet, in.config.Url, nil)
	req = req.WithContext(ctx)

	// Make misp client.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: in.config.ServerName,
			},
		},
	}

	// Start receiving messages.
	msg, err := client.Do(req)
	if err != nil {
		in.log.Debug("OnEvent returned false. Stopping input worker.")
		cancel()
	}
	responseData, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		in.log.Debug("Failed to read http message body")
		cancel()
	}
	if ok := in.outlet.OnEvent(makeEvent(string(responseData))); ok {
		return nil
	}
	in.log.Debug("OnEvent returned false. Stopping input worker.")
	cancel()
	return errors.New("OnEvent returned false")
}

// Stop stops the misp input and waits for it to fully stop.
func (in *mispInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *mispInput) Wait() {
	in.Stop()
}

func makeEvent(body string) beat.Event {
	id := uuid.New().String()

	fields := common.MapStr{
		"event": common.MapStr{
			"id":      id,
			"created": time.Now().UTC(),
		},
		"message": string(body),
	}
	// if len(msg.Attributes) > 0 {
	// 	fields.Put("labels", msg.Attributes)
	// }

	return beat.Event{
		Timestamp: time.Now().UTC(),
		Meta: common.MapStr{
			"id": id,
		},
		Fields: fields,
	}
}
