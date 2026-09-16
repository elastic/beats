// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package elasticsearchclient is an OpenTelemetry Collector extension that owns
// the lifecycle of a single Elasticsearch connection per configured instance.
package elasticsearchclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	_ extension.Extension = (*elasticsearchClient)(nil)

	// ErrNotStarted is returned by Request when Start has not created a client.
	ErrNotStarted = errors.New("elasticsearchclient extension is not started")
	// ErrShutdown is returned by Request after Shutdown.
	ErrShutdown = errors.New("elasticsearchclient extension is shut down")
)

// elasticsearchClient owns one non-thread-safe Elasticsearch connection.
// requestMu serializes requests, while lifecycleMu lets Shutdown cancel an
// in-flight request before waiting to close the client.
type elasticsearchClient struct {
	cfg    *Config
	logger *logp.Logger
	info   beat.Info

	lifecycleMu sync.Mutex
	requestMu   sync.Mutex
	client      *eslegclient.Connection
	cancel      context.CancelFunc
	starting    bool
	closed      bool
}

func (e *elasticsearchClient) Start(ctx context.Context, host component.Host) error {
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

	// Connection stores this context on every subsequent HTTP request, so use
	// an extension-owned context that is cancelled in Shutdown. While startup
	// is in progress, also cancel it if the Collector cancels Start's context.
	clientCtx, cancel := context.WithCancel(context.Background())
	stopStartCancellation := context.AfterFunc(ctx, cancel)

	e.lifecycleMu.Lock()
	if e.closed {
		e.lifecycleMu.Unlock()
		stopStartCancellation()
		cancel()
		return ErrShutdown
	}
	if e.starting || e.client != nil {
		e.lifecycleMu.Unlock()
		stopStartCancellation()
		cancel()
		return errors.New("elasticsearchclient extension is already started")
	}
	e.starting = true
	// Publish cancellation before connecting so Shutdown can interrupt the
	// initial Elasticsearch ping.
	e.cancel = cancel
	e.lifecycleMu.Unlock()

	client, err := newConnectedClient(clientCtx, e.cfg, e.info, e.logger)
	stopStartCancellation()
	if err != nil {
		cancel()
		if startErr := ctx.Err(); startErr != nil {
			return e.startFailed(host, startErr)
		}
		return e.startFailed(host, err)
	}
	// The initial ping may succeed just before Start is canceled. Do not publish
	// a client whose startup lifecycle was canceled after the ping completed.
	if err := ctx.Err(); err != nil {
		cancel()
		_ = client.Close()
		return e.startFailed(host, err)
	}

	e.lifecycleMu.Lock()
	if e.closed {
		e.starting = false
		e.lifecycleMu.Unlock()
		cancel()
		if closeErr := client.Close(); closeErr != nil {
			return fmt.Errorf("elasticsearchclient extension is shut down: failed closing unused client: %w", closeErr)
		}
		return ErrShutdown
	}

	e.starting = false
	e.client = client
	e.lifecycleMu.Unlock()
	componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
	return nil
}

// startFailed clears the cancellation function published for a failed Start.
func (e *elasticsearchClient) startFailed(host component.Host, connectErr error) error {
	e.lifecycleMu.Lock()
	closed := e.closed
	e.starting = false
	e.cancel = nil
	e.lifecycleMu.Unlock()

	if closed {
		return ErrShutdown
	}

	err := fmt.Errorf("failed connecting elasticsearch client: %w", connectErr)
	componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
	return err
}

// newConnectedClient adapts the extension's explicit endpoint and transport
// contract to the eslegclient settings.
func newConnectedClient(ctx context.Context, cfg *Config, info beat.Info, logger *logp.Logger) (*eslegclient.Connection, error) {
	parameters := cfg.Parameters
	if len(parameters) == 0 {
		parameters = nil
	}

	errs := make([]string, 0, len(cfg.Hosts))
	for _, host := range cfg.Hosts {
		esURL, err := common.MakeURL(cfg.Protocol, cfg.Path, host, 9200)
		if err != nil {
			errs = append(errs, fmt.Sprintf("invalid host %q: %v", host, err))
			continue
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
			errs = append(errs, fmt.Sprintf("creating client for %s: %v", esURL, err))
			continue
		}

		if err := client.Connect(ctx); err != nil {
			_ = client.Close()
			errs = append(errs, fmt.Sprintf("connecting to %s: %v", esURL, err))
			continue
		}
		return client, nil
	}

	return nil, fmt.Errorf("couldn't connect to any configured Elasticsearch hosts: %s", strings.Join(errs, "; "))
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
	client := e.client
	e.client = nil
	if cancel != nil {
		cancel()
	}
	e.lifecycleMu.Unlock()

	if client == nil {
		return nil
	}

	// Cancel has been called, so an in-flight HTTP request can
	// observe cancellation and release the request lock.
	// Wait on requestMu to ensure all in-flight requests finish/are cancelled
	// before closing the client.
	e.requestMu.Lock()
	defer e.requestMu.Unlock()
	err := client.Close()
	if err != nil {
		return fmt.Errorf("failed closing elasticsearch client: %w", err)
	}
	return nil
}

// Request forwards one Elasticsearch REST call through the owned connection.
//
// The signature matches eslegclient.Connection.Request.
// The underlying connection is not goroutine-safe; the
// entire call, including JSON encoding and response buffering, runs under the
// a request mutex. The returned body is copied so callers can use it
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
	if e.client == nil {
		e.lifecycleMu.Unlock()
		return 0, nil, ErrNotStarted
	}
	client := e.client
	e.lifecycleMu.Unlock()

	status, resp, err := client.Request(method, path, pipeline, params, body)
	return status, bytes.Clone(resp), err
}
