// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type DopplerCallbacks struct {
	Log    func(evt Event)
	Metric func(evt Event)
	Error  func(evt EventError)
}

type DopplerConsumer struct {
	sync.Mutex

	subscriptionID string
	callbacks      DopplerCallbacks
	consumer       *consumer.Consumer
	tokenRefresher consumer.TokenRefresher

	log     *logp.Logger
	wg      sync.WaitGroup
	stop    chan struct{}
	started bool
}

func newDopplerConsumer(address string, id string, log *logp.Logger, tlsConfig *tls.Config, proxy func(*http.Request) (*url.URL, error), tr *TokenRefresher, callbacks DopplerCallbacks) (*DopplerConsumer, error) {
	c := consumer.New(address, tlsConfig, proxy)
	c.RefreshTokenFrom(tr)
	c.SetDebugPrinter(newLogpDebugPrinter(log))

	return &DopplerConsumer{
		subscriptionID: id,
		consumer:       c,
		tokenRefresher: tr,
		callbacks:      callbacks,
		log:            log,
	}, nil
}

func (c *DopplerConsumer) Run() {
	c.Lock()
	defer c.Unlock()
	if c.started {
		return
	}
	c.stop = make(chan struct{})

	if c.callbacks.Log != nil {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.logsFirehose()
		}()
	}

	if c.callbacks.Metric != nil {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.metricsFirehose()
		}()
	}

	c.started = true
}

func (c *DopplerConsumer) logsFirehose() {
	c.firehose(c.callbacks.Log, consumer.LogMessages)
}

func (c *DopplerConsumer) metricsFirehose() {
	c.firehose(c.callbacks.Metric, consumer.Metrics)
}

func (c *DopplerConsumer) firehose(cb func(evt Event), filter consumer.EnvelopeFilter) {
	var msgChan <-chan *events.Envelope
	var errChan <-chan error
	filterFn := filterNoFilter
	if filter == consumer.LogMessages {
		// We are interested in more envelopes than the ones obtained when filtering
		// by log messages, retrieve them all and filter later.
		// If this causes performance or other problems, we will have to investigate
		// if it is possible to pass different filters to the firehose url.
		filterFn = filterLogs
		msgChan, errChan = c.consumer.Firehose(c.subscriptionID, "")
	} else {
		msgChan, errChan = c.consumer.FilteredFirehose(c.subscriptionID, "", filter)
	}
	for {
		select {
		case env := <-msgChan:
			if !filterFn(env) {
				continue
			}
			event := EnvelopeToEvent(env)
			if event == nil {
				c.log.Debugf("Envelope couldn't be converted to event: %+v", env)
				continue
			}
			if evtError, ok := event.(*EventError); ok {
				c.reportError(*evtError)
				continue
			}
			cb(event)
		case err := <-errChan:
			if err != nil {
				// This error is an error on the connection, not a cloud foundry
				// error envelope. Firehose should be able to reconnect, so just log it.
				c.log.Infof("Error received on firehose: %v", err)
			}
		case <-c.stop:
			return
		}
	}
}

func filterNoFilter(*events.Envelope) bool { return true }
func filterLogs(e *events.Envelope) bool {
	if e == nil || e.EventType == nil {
		return false
	}
	switch *e.EventType {
	case events.Envelope_HttpStartStop, events.Envelope_LogMessage, events.Envelope_Error:
		return true
	}
	return false
}

func (c *DopplerConsumer) reportError(e EventError) {
	if c.callbacks.Error == nil {
		c.log.Debugf("No callback for errors, error received: %s", e)
		return
	}
	c.callbacks.Error(e)
}

func (c *DopplerConsumer) Stop() {
	c.Lock()
	defer c.Unlock()
	if !c.started {
		return
	}

	close(c.stop)
	err := c.consumer.Close()
	if err != nil {
		c.log.Errorf("Error while closing doppler consumer: %v", err)
	}

	c.started = false
}

func (c *DopplerConsumer) Wait() {
	c.Stop()
	c.wg.Wait()
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

type LogpDebugPrinter struct {
	log *logp.Logger
}

func newLogpDebugPrinter(log *logp.Logger) *LogpDebugPrinter {
	return &LogpDebugPrinter{log: log}
}

var authorizationHeaderRE = regexp.MustCompile("Authorization: .*\n")

func (p *LogpDebugPrinter) Print(title, dump string) {
	if !p.log.IsDebug() {
		return
	}
	// Avoid printing out authorization tokens, Sec-WebSocket-Key is already hidden by the library.
	dump = authorizationHeaderRE.ReplaceAllString(dump, "Authorization: [HIDDEN]\n")
	p.log.Debugf("%s: %s", title, dump)
}
