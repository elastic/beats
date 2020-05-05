// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmhttp

import (
	"io"
	"net/http"
	"sync/atomic"
	"unsafe"

	"go.elastic.co/apm"
)

// WrapClient returns a new *http.Client with all fields copied
// across, and the Transport field wrapped with WrapRoundTripper
// such that client requests are reported as spans to Elastic APM
// if their context contains a sampled transaction.
//
// Spans are started just before the request is sent, and ended
// immediately if the request returned an error (e.g. due to socket
// timeout, but not a valid response with a non-200 status code),
// or otherwise when the response body is fully consumed or closed.
//
// If c is nil, then http.DefaultClient is wrapped.
func WrapClient(c *http.Client, o ...ClientOption) *http.Client {
	if c == nil {
		c = http.DefaultClient
	}
	copied := *c
	copied.Transport = WrapRoundTripper(copied.Transport, o...)
	return &copied
}

// WrapRoundTripper returns an http.RoundTripper wrapping r, reporting each
// request as a span to Elastic APM, if the request's context contains a
// sampled transaction.
//
// If r is nil, then http.DefaultTransport is wrapped.
func WrapRoundTripper(r http.RoundTripper, o ...ClientOption) http.RoundTripper {
	if r == nil {
		r = http.DefaultTransport
	}
	rt := &roundTripper{
		r:              r,
		requestName:    ClientRequestName,
		requestIgnorer: IgnoreNone,
	}
	for _, o := range o {
		o(rt)
	}
	return rt
}

type roundTripper struct {
	r              http.RoundTripper
	requestName    RequestNameFunc
	requestIgnorer RequestIgnorerFunc
}

// RoundTrip delegates to r.r, emitting a span if req's context
// contains a transaction.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if r.requestIgnorer(req) {
		return r.r.RoundTrip(req)
	}
	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return r.r.RoundTrip(req)
	}

	// RoundTrip is not supposed to mutate req, so copy req
	// and set the trace-context headers only in the copy.
	reqCopy := *req
	reqCopy.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		reqCopy.Header[k] = v
	}
	req = &reqCopy

	propagateLegacyHeader := tx.ShouldPropagateLegacyHeader()
	traceContext := tx.TraceContext()
	if !traceContext.Options.Recorded() {
		r.setHeaders(req, traceContext, propagateLegacyHeader)
		return r.r.RoundTrip(req)
	}

	name := r.requestName(req)
	span := tx.StartSpan(name, "external.http", apm.SpanFromContext(ctx))
	if !span.Dropped() {
		traceContext = span.TraceContext()
		ctx = apm.ContextWithSpan(ctx, span)
		req = RequestWithContext(ctx, req)
		span.Context.SetHTTPRequest(req)
	} else {
		span.End()
		span = nil
	}

	r.setHeaders(req, traceContext, propagateLegacyHeader)
	resp, err := r.r.RoundTrip(req)
	if span != nil {
		if err != nil {
			span.End()
		} else {
			span.Context.SetHTTPStatusCode(resp.StatusCode)
			resp.Body = &responseBody{span: span, body: resp.Body}
		}
	}
	return resp, err
}

func (r *roundTripper) setHeaders(req *http.Request, traceContext apm.TraceContext, propagateLegacyHeader bool) {
	headerValue := FormatTraceparentHeader(traceContext)
	if propagateLegacyHeader {
		req.Header.Set(ElasticTraceparentHeader, headerValue)
	}
	req.Header.Set(W3CTraceparentHeader, headerValue)
	if tracestate := traceContext.State.String(); tracestate != "" {
		req.Header.Set(TracestateHeader, tracestate)
	}
}

// CloseIdleConnections calls r.r.CloseIdleConnections if the method exists.
func (r *roundTripper) CloseIdleConnections() {
	type closeIdler interface {
		CloseIdleConnections()
	}
	if r, ok := r.r.(closeIdler); ok {
		r.CloseIdleConnections()
	}
}

// CancelRequest calls r.r.CancelRequest(req) if the method exists.
func (r *roundTripper) CancelRequest(req *http.Request) {
	type cancelRequester interface {
		CancelRequest(*http.Request)
	}
	if r, ok := r.r.(cancelRequester); ok {
		r.CancelRequest(req)
	}
}

type responseBody struct {
	span *apm.Span
	body io.ReadCloser
}

// Close closes the response body, and ends the span if it hasn't already been ended.
func (b *responseBody) Close() error {
	b.endSpan()
	return b.body.Close()
}

// Read reads from the response body, and ends the span when io.EOF is returend if
// the span hasn't already been ended.
func (b *responseBody) Read(p []byte) (n int, err error) {
	n, err = b.body.Read(p)
	if err == io.EOF {
		b.endSpan()
	}
	return n, err
}

func (b *responseBody) endSpan() {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&b.span))
	if old := atomic.SwapPointer(addr, nil); old != nil {
		(*apm.Span)(old).End()
	}
}

// ClientOption sets options for tracing client requests.
type ClientOption func(*roundTripper)

// WithClientRequestName returns a ClientOption which sets r as the function
// to use to obtain the span name for the given http request.
func WithClientRequestName(r RequestNameFunc) ClientOption {
	if r == nil {
		panic("r == nil")
	}

	return ClientOption(func(rt *roundTripper) {
		rt.requestName = r
	})
}
