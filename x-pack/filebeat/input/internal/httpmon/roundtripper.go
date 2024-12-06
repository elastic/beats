// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package httpmon provides http request and response metric monitoring.
package httpmon

import (
	"net/http"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

var _ http.RoundTripper = (*MetricsRoundTripper)(nil)

// MetricsRoundTripper is an http.RoundTripper that monitors requests and responses.
type MetricsRoundTripper struct {
	transport http.RoundTripper

	metrics *httpMetrics
}

type httpMetrics struct {
	reqs          *monitoring.Uint // total number of requests
	reqErrs       *monitoring.Uint // total number of request errors
	reqDelete     *monitoring.Uint // number of DELETE requests
	reqGet        *monitoring.Uint // number of GET requests
	reqHead       *monitoring.Uint // number of HEAD requests
	reqOptions    *monitoring.Uint // number of OPTIONS requests
	reqPatch      *monitoring.Uint // number of PATCH requests
	reqPost       *monitoring.Uint // number of POST requests
	reqPut        *monitoring.Uint // number of PUT requests
	reqsAccSize   *monitoring.Uint // accumulated request body size
	reqsSize      metrics.Sample   // histogram of the request body size
	resps         *monitoring.Uint // total number of responses
	respErrs      *monitoring.Uint // total number of response errors
	resp1xx       *monitoring.Uint // number of 1xx responses
	resp2xx       *monitoring.Uint // number of 2xx responses
	resp3xx       *monitoring.Uint // number of 3xx responses
	resp4xx       *monitoring.Uint // number of 4xx responses
	resp5xx       *monitoring.Uint // number of 5xx responses
	respsAccSize  *monitoring.Uint // accumulated response body size
	respsSize     metrics.Sample   // histogram of the response body size
	roundTripTime metrics.Sample   // histogram of the round trip (request -> response) time
}

// NewMetricsRoundTripper returns a MetricsRoundTripper that sends requests and
// responses metrics to the provided input monitoring registry.
// It will register all http related metrics into the provided registry, but it is not responsible
// for its lifecyle.
func NewMetricsRoundTripper(next http.RoundTripper, reg *monitoring.Registry) *MetricsRoundTripper {
	return &MetricsRoundTripper{
		transport: next,
		metrics:   newHTTPMetrics(reg),
	}
}

func newHTTPMetrics(reg *monitoring.Registry) *httpMetrics {
	if reg == nil {
		return nil
	}

	out := &httpMetrics{
		reqs:          monitoring.NewUint(reg, "http_request_total"),
		reqErrs:       monitoring.NewUint(reg, "http_request_errors_total"),
		reqDelete:     monitoring.NewUint(reg, "http_request_delete_total"),
		reqGet:        monitoring.NewUint(reg, "http_request_get_total"),
		reqHead:       monitoring.NewUint(reg, "http_request_head_total"),
		reqOptions:    monitoring.NewUint(reg, "http_request_options_total"),
		reqPatch:      monitoring.NewUint(reg, "http_request_patch_total"),
		reqPost:       monitoring.NewUint(reg, "http_request_post_total"),
		reqPut:        monitoring.NewUint(reg, "http_request_put_total"),
		reqsAccSize:   monitoring.NewUint(reg, "http_request_body_bytes_total"),
		reqsSize:      metrics.NewUniformSample(1024),
		resps:         monitoring.NewUint(reg, "http_response_total"),
		respErrs:      monitoring.NewUint(reg, "http_response_errors_total"),
		resp1xx:       monitoring.NewUint(reg, "http_response_1xx_total"),
		resp2xx:       monitoring.NewUint(reg, "http_response_2xx_total"),
		resp3xx:       monitoring.NewUint(reg, "http_response_3xx_total"),
		resp4xx:       monitoring.NewUint(reg, "http_response_4xx_total"),
		resp5xx:       monitoring.NewUint(reg, "http_response_5xx_total"),
		respsAccSize:  monitoring.NewUint(reg, "http_response_body_bytes_total"),
		respsSize:     metrics.NewUniformSample(1024),
		roundTripTime: metrics.NewUniformSample(1024),
	}

	_ = adapter.GetGoMetrics(reg, "http_request_body_bytes", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.reqsSize))
	_ = adapter.GetGoMetrics(reg, "http_response_body_bytes", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.respsSize))
	_ = adapter.GetGoMetrics(reg, "http_round_trip_time", adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(out.roundTripTime))

	return out
}

// RoundTrip implements the http.RoundTripper interface, sending
// request and response metrics to the underlying registry.
//
//	http_request_total
//	http_request_errors_total
//	http_request_delete_total
//	http_request_get_total
//	http_request_head_total
//	http_request_options_total
//	http_request_patch_total
//	http_request_post_total
//	http_request_put_total
//	http_request_body_bytes_total
//	http_request_body_bytes
//	http_response_total
//	http_response_errors_total
//	http_response_1xx_total
//	http_response_2xx_total
//	http_response_3xx_total
//	http_response_4xx_total
//	http_response_5xx_total
//	http_response_body_bytes_total
//	http_response_body_bytes
//	http_round_trip_time
func (rt *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.metrics == nil {
		return rt.transport.RoundTrip(req)
	}

	rt.metrics.reqs.Add(1)

	rt.monitorByMethod(req.Method)

	if req.ContentLength >= 0 {
		rt.metrics.reqsAccSize.Add(uint64(req.ContentLength))
		rt.metrics.reqsSize.Update(req.ContentLength)
	}

	reqStart := time.Now()
	resp, err := rt.transport.RoundTrip(req)
	rt.metrics.roundTripTime.Update(time.Since(reqStart).Nanoseconds())

	if resp != nil {
		rt.metrics.resps.Add(1)
	}

	if resp == nil || err != nil {
		rt.metrics.respErrs.Add(1)
		return resp, err
	}

	rt.monitorByStatusCode(resp.StatusCode)

	if resp.ContentLength >= 0 {
		rt.metrics.respsAccSize.Add(uint64(resp.ContentLength))
		rt.metrics.respsSize.Update(resp.ContentLength)
	}

	return resp, err
}

func (rt *MetricsRoundTripper) monitorByMethod(method string) {
	switch method {
	case http.MethodDelete:
		rt.metrics.reqDelete.Add(1)
	case http.MethodGet:
		rt.metrics.reqGet.Add(1)
	case http.MethodHead:
		rt.metrics.reqHead.Add(1)
	case http.MethodOptions:
		rt.metrics.reqOptions.Add(1)
	case http.MethodPatch:
		rt.metrics.reqPatch.Add(1)
	case http.MethodPost:
		rt.metrics.reqPost.Add(1)
	case http.MethodPut:
		rt.metrics.reqPut.Add(1)
	}
}

func (rt *MetricsRoundTripper) monitorByStatusCode(code int) {
	switch code / 100 {
	case 1:
		rt.metrics.resp1xx.Add(1)
	case 2:
		rt.metrics.resp2xx.Add(1)
	case 3:
		rt.metrics.resp3xx.Add(1)
	case 4:
		rt.metrics.resp4xx.Add(1)
	case 5:
		rt.metrics.resp5xx.Add(1)
	}
}
