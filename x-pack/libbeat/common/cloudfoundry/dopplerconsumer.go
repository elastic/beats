// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"net/http"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
)

type DopplerCallbacks struct {
	Log    func(evt Event)
	Metric func(evt Event)
	Error  func(evt EventError)
}

type DopplerConsumer struct {
	subscriptionID string
	callbacks      DopplerCallbacks
	consumer       *consumer.Consumer
	tokenRefresher consumer.TokenRefresher

	stop chan struct{}
}

func newDopplerConsumer(address string, id string, client *http.Client, tr *TokenRefresher, callbacks DopplerCallbacks) (*DopplerConsumer, error) {
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("expected http transport on client")
	}

	c := consumer.New(address, transport.TLSClientConfig, transport.Proxy)
	c.RefreshTokenFrom(tr)

	return &DopplerConsumer{
		subscriptionID: id,
		consumer:       c,
		tokenRefresher: tr,
		callbacks:      callbacks,
	}, nil
}

func (c *DopplerConsumer) Run() {
	// FIXME: ensure it is not run twice
	c.stop = make(chan struct{})

	if c.callbacks.Log != nil {
		go c.logsFirehose()
	}

	if c.callbacks.Metric != nil {
		go c.metricsFirehose()
	}
}

func (c *DopplerConsumer) logsFirehose() {
	c.firehose(c.callbacks.Log, consumer.LogMessages)
}

func (c *DopplerConsumer) metricsFirehose() {
	c.firehose(c.callbacks.Metric, consumer.Metrics)
}

func (c *DopplerConsumer) firehose(cb func(evt Event), filter consumer.EnvelopeFilter) {
	// FIXME: Get the initial token from Run()
	token, _ := c.tokenRefresher.RefreshAuthToken()
	// FIXME: handle error
	msgChan, errChan := c.consumer.FilteredFirehose(c.subscriptionID, token, filter)
	for {
		select {
		case env := <-msgChan:
			event := envelopeToEvent(env)
			if event == nil {
				// FIXME: log unknown event
				continue
			}
			if evtError, ok := event.(*EventError); ok {
				c.reportError(*evtError)
				continue
			}
			cb(event)
		case _ = <-errChan:
			// FIXME: handle/log connection error
		case <-c.stop:
			return
		}
	}
}

func (c *DopplerConsumer) reportError(e EventError) {
	if c.callbacks.Error == nil {
		// FIXME: log at debug level
		return
	}
	c.callbacks.Error(e)
}

func (c *DopplerConsumer) Stop() {
	close(c.stop)
	err := c.consumer.Close()
	if err != nil {
		// FIXME: log
	}
}

type TokenRefresher struct {
	client *cfclient.Client
}

func TokenRefresherFromCfClient(c *cfclient.Client) *TokenRefresher {
	return &TokenRefresher{client: c}
}

func (tr *TokenRefresher) RefreshAuthToken() (token string, authError error) {
	return tr.client.GetToken()
}
