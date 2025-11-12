// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.elastic.co/apm/module/apmelasticsearch/v2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

var _ extensionauth.HTTPClient = (*authenticator)(nil)
var _ extensionauth.GRPCClient = (*authenticator)(nil)
var _ extension.Extension = (*authenticator)(nil)

// roundTripperProvider is an interface that provides a RoundTripper
type roundTripperProvider interface {
	RoundTripper() http.RoundTripper
}

type authenticator struct {
	telemetry  component.TelemetrySettings
	logger     *logp.Logger
	cfg        *Config
	rtProvider roundTripperProvider
}

func newAuthenticator(cfg *Config, telemetry component.TelemetrySettings) (*authenticator, error) {
	// logp.NewZapLogger essentially never returns an error; look within the implementation
	logger, err := logp.NewZapLogger(telemetry.Logger)
	if err != nil {
		return nil, err
	}

	auth := &authenticator{
		cfg:       cfg,
		telemetry: telemetry,
		logger:    logger,
	}
	return auth, nil
}

func (a *authenticator) Start(_ context.Context, host component.Host) error {
	switch {
	case a == nil:
		return fmt.Errorf("authenticator is nil")
	case a.cfg == nil:
		return fmt.Errorf("authenticator config is nil")
	}

	var provider roundTripperProvider
	prov, err := getHttpClient(a)
	if err != nil {
		componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
		err = fmt.Errorf("failed creating http client: %w", err)
		if !a.cfg.ContinueOnError {
			return err
		}
		a.logger.Warnf("%s", err.Error())
		provider = &errorRoundTripperProvider{err: err}
	} else {
		componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
		provider = prov
	}

	a.rtProvider = provider
	return nil
}

func (a *authenticator) Shutdown(_ context.Context) error {
	return nil
}

func (a *authenticator) RoundTripper(_ http.RoundTripper) (http.RoundTripper, error) {
	if a.rtProvider == nil {
		return nil, fmt.Errorf("authenticator not started")
	}
	return a.rtProvider.RoundTripper(), nil
}

// getHTTPOptions returns a list of http transport options
// these options are derived from beats codebase Ref: https://github.com/elastic/beats/blob/4dfef8b/libbeat/esleg/eslegclient/connection.go#L163-L171
// httpcommon.WithIOStats(s.Observer) is omitted as we do not have access to observer here
// httpcommon.WithHeaderRoundTripper with user-agent is also omitted as we continue to use ES exporter's user-agent
func (a *authenticator) getHTTPOptions(idleConnTimeout time.Duration) []httpcommon.TransportOption {
	return []httpcommon.TransportOption{
		httpcommon.WithLogger(a.logger),
		httpcommon.WithKeepaliveSettings{IdleConnTimeout: idleConnTimeout},
		httpcommon.WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
			return apmelasticsearch.WrapRoundTripper(rt)
		}),
	}
}

func (a *authenticator) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	// Elasticsearch doesn't support gRPC, this function won't be called
	return nil, nil
}

func getHttpClient(a *authenticator) (roundTripperProvider, error) {
	parsedCfg, err := config.NewConfigFrom(a.cfg.BeatAuthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed creating config: %w", err)
	}

	beatAuthConfig := ESAuthConfig{}
	err = parsedCfg.Unpack(&beatAuthConfig)
	if err != nil {
		return nil, fmt.Errorf("failed unpacking config: %w", err)
	}

	client, err := beatAuthConfig.Transport.Client(a.getHTTPOptions(beatAuthConfig.Transport.IdleConnTimeout)...)
	if err != nil {
		return nil, fmt.Errorf("failed creating http client: %w", err)
	}

	// if loadbalance is disabled, all outgoing requests should go to a single endpoint
	// unless the endpoint is unreachable, then the next endpoint in the list is used
	if !beatAuthConfig.LoadBalance {
		singleRouterProvider, err := NewSingleRouterProvider(beatAuthConfig, client)
		if err != nil {
			return nil, fmt.Errorf("failed creating http client: %w", err)
		}
		return singleRouterProvider, nil
	}

	return &httpClientProvider{client: client}, nil
}

// httpClientProvider provides a RoundTripper from an http.Client
type httpClientProvider struct {
	client *http.Client
}

func (h *httpClientProvider) RoundTripper() http.RoundTripper {
	return h.client.Transport
}

type singleRouterProvider struct {
	endpoints []*url.URL
	active    int
	client    *http.Client
	mx        sync.Mutex
}

// NewSingleRouterProvider returns a RoundTripper that atmost one active endpoint
// If the connection to the active client fails, the next endpoint is used and so
func NewSingleRouterProvider(config ESAuthConfig, client *http.Client) (*singleRouterProvider, error) {
	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("atleast one endpoint must be provided when loadbalance is disabled")
	}

	urls := make([]*url.URL, 0, len(config.Endpoints))
	for _, endpoint := range config.Endpoints {
		finalEndpoint, err := common.MakeURL(config.Protocol, config.Path, endpoint, 9200)
		if err != nil {
			return nil, fmt.Errorf("failed building URL for endpoint %q: %w", endpoint, err)
		}
		parsedEndpoint, err := url.Parse(finalEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed parsing endpoint %q: %w", endpoint, err)
		}
		urls = append(urls, parsedEndpoint)
	}

	return &singleRouterProvider{client: client, endpoints: urls, active: getNextActiveClient(-1, len(urls))}, nil
}

func (srp *singleRouterProvider) RoundTripper() http.RoundTripper {
	return srp
}

func (srp *singleRouterProvider) RoundTrip(req *http.Request) (*http.Response, error) {
	// use the first endpoint in the list
	srp.mx.Lock()
	req.URL = srp.endpoints[srp.active]
	srp.mx.Unlock()

	// perform the request
	resp, err := srp.client.Transport.RoundTrip(req)

	if err != nil && len(srp.endpoints) > 1 {
		// if response is unsuccessful, get a random next endpoint
		srp.mx.Lock()
		srp.active = getNextActiveClient(srp.active, len(srp.endpoints))
		srp.mx.Unlock()
	}

	// return the error as is for the caller of RoundTrip to handle retry logic
	return resp, err
}

// getNextActiveClient returns the next active client index given the current active index and total clients
// Note: This logic has been adapted from failoverClient in libbeat/outputs/failover.go
func getNextActiveClient(active int, totalClients int) (next int) {
	switch {
	case totalClients == 1:
		next = 0
	case totalClients == 2 && 0 <= active && active <= 1:
		next = 1 - active
	default:
		for {
			// Connect to random server to potentially spread the
			// load when large number of beats with same set of sinks
			// are started up at about the same time.
			next = rand.Int() % totalClients
			if next != active {
				break
			}
		}
	}

	return next
}

// errorRoundTripperProvider provides a RoundTripper that always returns an error
type errorRoundTripperProvider struct {
	err error
}

func (e *errorRoundTripperProvider) RoundTripper() http.RoundTripper {
	return &errorRoundTripper{err: e.err}
}

// errorRoundTripper is a RoundTripper that always returns an error
type errorRoundTripper struct {
	err error
}

func (e *errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}
