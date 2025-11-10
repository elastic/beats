// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"context"
	"fmt"
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

	if !beatAuthConfig.LoadBalance {
		loadBalanceProvider, err := NewLoadBalanceProvider(beatAuthConfig, client)
		if err != nil {
			return nil, fmt.Errorf("failed creating http client :%w", err)
		}
		return loadBalanceProvider, nil
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

// loadBalanceProvider provides a RoundTripper from an http.Client
type loadBalanceProvider struct {
	endpoints []*url.URL
	client    *http.Client
	mx        sync.Mutex
}

func NewLoadBalanceProvider(config ESAuthConfig, client *http.Client) (*loadBalanceProvider, error) {
	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("endpoints must be provided when load_balance is disabled")
	}

	urls := make([]*url.URL, 0, len(config.Endpoints))
	for i, endpoint := range config.Endpoints {
		finalEndpoint, err := common.MakeURL(config.Protocol, config.Path, endpoint, 9200)
		if err != nil {
			return nil, fmt.Errorf("failed building URL for endpoint %q: %w", endpoint, err)
		}
		urls[i], err = url.Parse(finalEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed parsing endpoint %q: %w", endpoint, err)
		}
	}

	return &loadBalanceProvider{client: client, endpoints: urls}, nil
}

func (lbr *loadBalanceProvider) RoundTripper() http.RoundTripper {
	return lbr
}

func (lbr *loadBalanceProvider) RoundTrip(req *http.Request) (*http.Response, error) {
	// use the first endpoint in the list
	lbr.mx.Lock()
	req.URL = lbr.endpoints[0]
	lbr.mx.Unlock()

	// perform the request
	resp, err := lbr.client.Transport.RoundTrip(req)

	if err != nil && len(lbr.endpoints) > 1 {
		// if response is unsuccessful, move the first endpoint to the end of the list
		lbr.mx.Lock()
		lbr.endpoints = append(lbr.endpoints[1:], lbr.endpoints[0])
		lbr.mx.Unlock()
	}

	// return the error as is for the caller of RoundTrip to handle retry logic
	return resp, err
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
