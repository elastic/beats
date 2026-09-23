// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package elasticsearchclient is an OpenTelemetry Collector extension that owns
// the lifecycle of Elasticsearch connections per configured instance.
package elasticsearchclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	_ extension.Extension = (*elasticsearchClient)(nil)

	// ErrNotStarted is returned by Request before Start has initialized the extension.
	ErrNotStarted = errors.New("elasticsearchclient extension is not started")
	// ErrShutdown is returned by Request after Shutdown.
	ErrShutdown = errors.New("elasticsearchclient extension is shut down")
)

const (
	reconnectBackoffInitial = time.Second
	reconnectBackoffMax     = time.Minute
)

// elasticsearchClient owns non-thread-safe Elasticsearch connections.
// requestMu serializes connection and request operations, while lifecycleMu
// lets Shutdown cancel an in-flight request before waiting to close clients.
type elasticsearchClient struct {
	cfg    *Config
	logger *logp.Logger
	info   beat.Info

	lifecycleMu      sync.Mutex
	requestMu        sync.Mutex
	host             component.Host
	candidateClients []*eslegclient.Connection
	client           *eslegclient.Connection
	ctx              context.Context
	cancel           context.CancelFunc
	connectBackoff   backoff.Backoff
	reconnectPending bool
	started          bool
	closed           bool
}

func (e *elasticsearchClient) Start(_ context.Context, host component.Host) error {
	if e == nil {
		return errors.New("elasticsearchclient extension is nil")
	}
	if e.cfg == nil {
		err := errors.New("elasticsearchclient config is nil")
		componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
		return err
	}

	if err := e.cfg.Validate(); err != nil {
		err = fmt.Errorf("invalid elasticsearchclient configuration: %w", err)
		componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
		return err
	}

	candidateClients, err := newCandidateClients(e.cfg, e.info, e.logger)
	if err != nil {
		err = fmt.Errorf("invalid elasticsearchclient configuration: %w", err)
		componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
		return err
	}

	// Connection stores this context on every subsequent HTTP request. It is
	// owned by the extension rather than Start because Collector Start contexts
	// end after initialization, while requests continue until Shutdown.
	clientCtx, cancel := context.WithCancel(context.Background())

	e.lifecycleMu.Lock()
	if e.closed {
		e.lifecycleMu.Unlock()
		cancel()
		closeClients(candidateClients)
		return ErrShutdown
	}
	if e.started {
		e.lifecycleMu.Unlock()
		cancel()
		closeClients(candidateClients)
		return errors.New("elasticsearchclient extension is already started")
	}
	e.started = true
	e.host = host
	e.candidateClients = candidateClients
	e.ctx = clientCtx
	e.cancel = cancel
	if e.connectBackoff == nil {
		e.connectBackoff = backoff.NewExpBackoff(reconnectBackoffInitial, reconnectBackoffMax)
	}
	e.reconnectPending = false
	e.lifecycleMu.Unlock()

	componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
	return nil
}

// newCandidateClients adapts the extension's explicit endpoint and transport
// contract to eslegclient settings without connecting. Since connections are
// lazy, Start cannot select a reachable host, so one candidate is retained
// per configured host for failover and reconnect; only one becomes active.
func newCandidateClients(cfg *Config, info beat.Info, logger *logp.Logger) ([]*eslegclient.Connection, error) {
	parameters := cfg.Parameters
	if len(parameters) == 0 {
		parameters = nil
	}

	candidateClients := make([]*eslegclient.Connection, 0, len(cfg.Hosts))
	for _, host := range cfg.Hosts {
		esURL, err := common.MakeURL(cfg.Protocol, cfg.Path, host, 9200)
		if err != nil {
			_ = closeClients(candidateClients)
			return nil, fmt.Errorf("invalid host %q: %w", host, err)
		}

		client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
			URL:             esURL,
			Beatname:        info.Beat,
			UserAgent:       info.UserAgent,
			Username:        cfg.Username,
			Password:        cfg.Password,
			APIKey:          cfg.APIKey,
			Headers:         cfg.Headers,
			Parameters:      parameters,
			Transport:       cfg.Transport,
			IdleConnTimeout: cfg.Transport.IdleConnTimeout,
		}, logger)
		if err != nil {
			_ = closeClients(candidateClients)
			return nil, fmt.Errorf("creating client for %s: %w", esURL, err)
		}
		candidateClients = append(candidateClients, client)
	}

	return candidateClients, nil
}

func (e *elasticsearchClient) Shutdown(_ context.Context) error {
	if e == nil {
		return nil
	}

	e.lifecycleMu.Lock()
	if e.closed {
		e.lifecycleMu.Unlock()
		return nil
	}
	e.closed = true

	cancel := e.cancel
	e.cancel = nil
	candidateClients := e.candidateClients
	e.candidateClients = nil
	e.client = nil
	e.ctx = nil
	e.reconnectPending = false
	if cancel != nil {
		cancel()
	}
	e.lifecycleMu.Unlock()

	if len(candidateClients) == 0 {
		return nil
	}

	// Cancel has been called, so an in-flight HTTP request can
	// observe cancellation and release the request lock.
	// Wait on requestMu to ensure all in-flight requests finish/are cancelled
	// before closing the client.
	e.requestMu.Lock()
	defer e.requestMu.Unlock()
	if err := closeClients(candidateClients); err != nil {
		return fmt.Errorf("failed closing elasticsearch client: %w", err)
	}
	return nil
}

// Request forwards one Elasticsearch REST call through the owned connection.
//
// The signature matches eslegclient.Connection.Request.
// The underlying connection is not goroutine-safe; the
// entire call, including JSON encoding and response buffering, runs under the
// request mutex. The returned body is copied so callers can use it
// after Request returns.
func (e *elasticsearchClient) Request(
	method, path, pipeline string,
	params map[string]string,
	body any,
) (int, []byte, error) {

	if e == nil {
		return 0, nil, ErrNotStarted
	}

	// Acquire requestMu first so we won't race with shutdown
	e.requestMu.Lock()
	defer e.requestMu.Unlock()

	e.lifecycleMu.Lock()
	if e.closed {
		e.lifecycleMu.Unlock()
		return 0, nil, ErrShutdown
	}
	if !e.started {
		e.lifecycleMu.Unlock()
		return 0, nil, ErrNotStarted
	}
	client := e.client
	candidateClients := e.candidateClients
	clientCtx := e.ctx
	host := e.host
	e.lifecycleMu.Unlock()

	if client == nil {
		var err error
		client, err = e.connect(candidateClients, clientCtx)
		if err != nil {
			if errors.Is(err, ErrShutdown) {
				return 0, nil, err
			}
			componentstatus.ReportStatus(host, componentstatus.NewRecoverableErrorEvent(err))
			return 0, nil, err
		}
	}

	status, resp, err := client.Request(method, path, pipeline, params, body)
	if status == 0 && err != nil {
		e.markDisconnected(client)
	}
	if (status == 0 && err != nil) || status >= 500 {
		requestErr := err
		if requestErr == nil {
			requestErr = fmt.Errorf("elasticsearch returned HTTP %d", status)
		}
		if !e.isClosed() {
			componentstatus.ReportStatus(
				host,
				componentstatus.NewRecoverableErrorEvent(
					fmt.Errorf("elasticsearch request failed: %w", requestErr),
				),
			)
		}
	} else if err == nil {
		componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
	}

	return status, bytes.Clone(resp), err
}

// connect selects the first reachable configured host. It is called while
// requestMu is held, so Connect and the eventual Request never race with each
// other or with another reconnect attempt.
func (e *elasticsearchClient) connect(candidateClients []*eslegclient.Connection, ctx context.Context) (*eslegclient.Connection, error) {
	e.lifecycleMu.Lock()
	reconnectPending := e.reconnectPending
	connectBackoff := e.connectBackoff
	e.lifecycleMu.Unlock()

	// The first lazy connection must be attempted immediately. After any
	// connection or transport failure, wait before the next attempt so callers
	// do not repeatedly dial unavailable hosts.
	if reconnectPending && !connectBackoff.Wait(ctx) {
		return nil, ErrShutdown
	}

	errs := make([]string, 0, len(candidateClients))
	for _, client := range candidateClients {
		if err := client.Connect(ctx); err != nil {
			if e.isClosed() {
				return nil, ErrShutdown
			}
			errs = append(errs, fmt.Sprintf("connecting to %s: %v", client.URL, err))
			continue
		}

		e.lifecycleMu.Lock()
		if e.closed {
			e.lifecycleMu.Unlock()
			return nil, ErrShutdown
		}
		e.client = client
		e.reconnectPending = false
		connectBackoff.Reset()
		e.lifecycleMu.Unlock()
		return client, nil
	}
	err := fmt.Errorf("couldn't connect to any configured Elasticsearch hosts: %s", strings.Join(errs, "; "))
	e.lifecycleMu.Lock()
	if !e.closed {
		e.reconnectPending = true
	}
	e.lifecycleMu.Unlock()
	return nil, err
}

func (e *elasticsearchClient) markDisconnected(client *eslegclient.Connection) {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	if !e.closed && e.client == client {
		e.client = nil
		e.reconnectPending = true
	}
}

func (e *elasticsearchClient) isClosed() bool {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	return e.closed
}

func closeClients(clients []*eslegclient.Connection) error {
	var errs []string
	for _, client := range clients {
		if err := client.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
