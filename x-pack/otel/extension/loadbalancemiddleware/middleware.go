// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadbalancemiddleware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

// LoadBalanceMiddleware is an extension that returns a single endpoint from a list of configured endpoints
// If the connection is unsuccessful, it returns the next configured endpoint and so on.
// This is in essence supporting `output.[type].loadbalance:false` functionality from Beats in OTEL Collector.
var _ extensionmiddleware.HTTPClient = (*loadBalanceMiddleware)(nil)

// roundTripperProvider is an interface that provides a RoundTripper
type roundTripperProvider interface {
	RoundTripper(base http.RoundTripper) http.RoundTripper
}

type loadBalanceMiddleware struct {
	cfg      *Config
	provider roundTripperProvider
}

func newLoadBalanceMiddleware(cfg *Config, _ component.TelemetrySettings) (extension.Extension, error) {
	return &loadBalanceMiddleware{
		cfg: cfg,
	}, nil
}

func (lb *loadBalanceMiddleware) Start(ctx context.Context, host component.Host) error {
	switch {
	case lb == nil:
		return fmt.Errorf("loadbalance middlewar is nil")
	case lb.cfg == nil:
		return fmt.Errorf("loadbalance middleware config is nil")
	case len(lb.cfg.Endpoints) == 0:
		return fmt.Errorf("loadbalance middleware endpoints cannot be empty")
	}

	// build list of URLs from endpoints
	urls := make([]*url.URL, 0, len(lb.cfg.Endpoints))
	for i, endpoint := range lb.cfg.Endpoints {
		finalEndpoint, err := common.MakeURL(lb.cfg.Protocol, lb.cfg.Path, endpoint, 9200)
		if err != nil {
			err = fmt.Errorf("failed building URL for endpoint %q: %w", endpoint, err)
			componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
			if !lb.cfg.ContinueOnError {
				return err
			}
			lb.provider = &errorRoundTripperProvider{err: err}
			return nil
		}
		urls[i], err = url.Parse(finalEndpoint)
		if err != nil {
			err = fmt.Errorf("failed parsing endpoint %q: %w", endpoint, err)
			componentstatus.ReportStatus(host, componentstatus.NewPermanentErrorEvent(err))
			if !lb.cfg.ContinueOnError {
				return err
			}
			lb.provider = &errorRoundTripperProvider{err: err}
			return nil
		}
	}
	lb.provider = &loadBalanceRoundTripperProvider{endpoints: urls}
	return nil
}

func (lb *loadBalanceMiddleware) Shutdown(ctx context.Context) error {
	return nil
}

func (lb *loadBalanceMiddleware) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return lb.provider.RoundTripper(base), nil
}

type loadBalanceRoundTripperProvider struct {
	endpoints []*url.URL
	base      http.RoundTripper
	mx        sync.Mutex
}

func (lbr *loadBalanceRoundTripperProvider) RoundTripper(base http.RoundTripper) http.RoundTripper {
	// we assume base remains constant for the lifetime of the provider
	lbr.base = base
	return lbr
}

func (lbr *loadBalanceRoundTripperProvider) RoundTrip(req *http.Request) (*http.Response, error) {
	// use the first endpoint in the list
	lbr.mx.Lock()
	req.URL = lbr.endpoints[0]
	lbr.mx.Unlock()
	// perform the request
	resp, err := lbr.base.RoundTrip(req)
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

func (e *errorRoundTripperProvider) RoundTripper(_ http.RoundTripper) http.RoundTripper {
	return &errorRoundTripper{err: e.err}
}

// errorRoundTripper is a RoundTripper that always returns an error
type errorRoundTripper struct {
	err error
}

func (e *errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}
