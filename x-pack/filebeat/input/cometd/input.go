// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	cometdVersion     = "38.0"
	inputName         = "cometd"
	subscribe_channel = "/meta/subscribe"
	connetion_type    = "long-polling"
	connect_channel   = "/meta/connect"
	// Replay accepts the following values
	// -2: replay all events from past 24 hrs
	// -1: start at current
	// >= 0: start from this event number
	Replay = -1
)

var (
	st = status{[]string{}, "", false}
	wg sync.WaitGroup
)

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *cometdInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("Input worker has started.")
			defer in.log.Info("Input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

func (in *cometdInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()
	b := bayeux{}
	creds, err := in.config.Auth.OAuth2.client()
	if err != nil {
		return fmt.Errorf("error while getting Salesforce credentials: %v", err)
	}
	in.out, err = b.TopicToChannel(ctx, creds, in.config.ChannelName, in.log, in.out)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %v", err)
	}

	var event Event
	for e := range in.out {
		if !e.Successful {
			// To handle the last response where the object received was empty
			if e.Data.Object == nil {
				return nil
			}

			// Convert json.RawMessage response to []byte
			msg, err := json.Marshal(e.Data.Object)
			if err != nil {
				return fmt.Errorf("JSON error: %v", err)
			}

			// Extract event IDs from json.RawMessage
			err = json.Unmarshal(e.Data.Object, &event)
			if err != nil {
				return fmt.Errorf("error while parsing JSON: %v", err)
			}
			if ok := in.outlet.OnEvent(makeEvent(event.EventId, string(msg))); !ok {
				in.log.Debug("OnEvent returned false. Stopping input worker.")
				close(in.out)
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
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
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

	// Creating a new channel for cometd input
	in.out = make(chan triggerEvent)

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
	close(in.out)
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

	ackedCount *atomic.Uint32                   // Total number of successfully ACKed messages.
	Transport  httpcommon.HTTPTransportSettings `config:",inline"`
	Retry      retryConfig                      `config:"retry"`
	out        chan triggerEvent
}

// triggerEvent describes an event received from Bayeaux Endpoint
type triggerEvent struct {
	Channel  string `json:"channel"`
	ClientID string `json:"clientId"`
	Data     struct {
		Event struct {
			CreatedDate time.Time `json:"createdDate"`
			ReplayID    int       `json:"replayId"`
			Type        string    `json:"type"`
		} `json:"event"`
		Object json.RawMessage `json:"payload"`
	} `json:"data,omitempty"`
	Successful bool `json:"successful,omitempty"`
}

// status is the state of success and subscribed channels
type status struct {
	channels  []string
	clientID  string
	connected bool
}

type bayeuxHandshake []struct {
	ClientID string `json:"clientId"`
	Channel  string `json:"channel"`
	Ext      struct {
		Replay bool `json:"replay"`
	} `json:"ext"`
	MinimumVersion           string   `json:"minimumVersion"`
	Successful               bool     `json:"successful"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
	Version                  string   `json:"version"`
}

type subscription struct {
	ClientID     string `json:"clientId"`
	Channel      string `json:"channel"`
	Subscription string `json:"subscription"`
	Successful   bool   `json:"successful"`
}

type clientIDAndCookies struct {
	clientID string
	cookies  []*http.Cookie
}

// bayeux struct allow for centralized storage of creds, ids, and cookies
type bayeux struct {
	creds  credentials
	id     clientIDAndCookies
	client *http.Client
}

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

type Event struct {
	EventId string `json:"EventIdentifier"`
}

func (c credentials) bayeuxUrl() string {
	return c.InstanceURL + "/cometd/" + cometdVersion
}

// Call is the base function for making bayeux requests
func (b *bayeux) call(body string, route string) (*http.Response, error) {
	var jsonStr = []byte(body)
	req, err := http.NewRequest("POST", route, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("bad Call request: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.creds.AccessToken))
	// Passing back cookies is required though undocumented in Salesforce API
	// SF Reference: https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_client_specs.htm
	for _, cookie := range b.id.cookies {
		req.AddCookie(cookie)
	}

	resp, err := b.client.Do(req)
	if err == io.EOF {
		return nil, fmt.Errorf("bad bayeuxCall io.EOF: %v", err)
	} else if err != nil {
		return nil, fmt.Errorf("unknown error: %v", err)
	}
	return resp, nil
}

func (b *bayeux) getClientID() error {
	handshake := `{"channel": "/meta/handshake", "supportedConnectionTypes": ["long-polling"], "version": "1.0"}`
	// Stub out clientIDAndCookies for first bayeuxCall
	resp, err := b.call(handshake, b.creds.bayeuxUrl())
	if err != nil {
		return fmt.Errorf("cannot get client id: %v", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var h bayeuxHandshake
	if err := decoder.Decode(&h); err == io.EOF {
		return fmt.Errorf("reached end of response: %v", err)
	} else if err != nil {
		return fmt.Errorf("error while reading response: %v", err)
	}
	creds := clientIDAndCookies{h[0].ClientID, resp.Cookies()}
	b.id = creds
	return nil
}

func (b *bayeux) subscribe(topic string, Replay int, log *logp.Logger) (subscription, error) {
	handshake := fmt.Sprintf(`{
								"channel": "%s",
								"subscription": "%s",
								"clientId": "%s",
								"ext": {
									"replay": {"%s": "%d"}
									}
								}`, subscribe_channel, topic, b.id.clientID, topic, Replay)
	resp, err := b.call(handshake, b.creds.bayeuxUrl())
	if err != nil {
		return subscription{}, fmt.Errorf("error while subscribing: %v", err)
	}

	defer resp.Body.Close()

	// Read the content
	var content []byte
	if resp.Body != nil {
		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return subscription{}, fmt.Errorf("error while reading content: %v", err)
		}
	}
	// Restore the io.ReadCloser to its original state
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(content))

	if resp.StatusCode > 299 {
		return subscription{}, fmt.Errorf("received non 2XX response: HTTP_CODE %v", resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)
	var h []subscription
	if err := decoder.Decode(&h); err == io.EOF {
		return subscription{}, fmt.Errorf("reached end of response: %v", err)
	} else if err != nil {
		return subscription{}, fmt.Errorf("error while reading response: %v", err)
	}
	sub := h[0]
	st.connected = sub.Successful
	st.clientID = sub.ClientID
	st.channels = append(st.channels, topic)
	log.Infof("Established connection(s): %+v", st)
	return sub, nil
}

func (b *bayeux) connect(out chan triggerEvent, log *logp.Logger) (chan triggerEvent, error) {
	go func() {
		for {
			postBody := fmt.Sprintf(`{"channel": "%s", "connectionType": "%s", "clientId": "%s"} `, connect_channel, connetion_type, b.id.clientID)
			resp, err := b.call(postBody, b.creds.bayeuxUrl())
			if err != nil {
				log.Warnf("Cannot connect to bayeux %s, trying again...", err)
			} else {
				// Read the content
				var b []byte
				if resp.Body != nil {
					b, err = ioutil.ReadAll(resp.Body)
				}
				if err != nil {
					return
				}
				// Restore the io.ReadCloser to its original state
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
				var x []triggerEvent
				decoder := json.NewDecoder(resp.Body)
				if err := decoder.Decode(&x); err != nil && err != io.EOF {
					return
				}
				for _, e := range x {
					out <- e
				}
			}
		}
	}()
	return out, nil
}

func (b *bayeux) TopicToChannel(ctx context.Context, creds credentials, topic string, log *logp.Logger, out chan triggerEvent) (chan triggerEvent, error) {
	b.creds = creds
	b.client = &http.Client{}
	err := b.getClientID()
	if err != nil {
		return make(chan triggerEvent), fmt.Errorf("error while getting client ID: %v", err)
	}
	b.subscribe(topic, Replay, log)
	c, err := b.connect(out, log)
	if err != nil {
		return make(chan triggerEvent), fmt.Errorf("error while creating a connection: %v", err)
	}
	wg.Add(1)
	return c, nil
}

func makeEvent(id string, body string) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"event": common.MapStr{
				"id":      id,
				"created": time.Now().UTC(),
			},
			"message": body,
		},
		Private: body,
	}
	event.SetID(id)

	return event
}
