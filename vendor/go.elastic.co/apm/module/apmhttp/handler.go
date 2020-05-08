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
	"context"
	"net/http"

	"go.elastic.co/apm"
)

// Wrap returns an http.Handler wrapping h, reporting each request as
// a transaction to Elastic APM.
//
// By default, the returned Handler will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
//
// By default, the returned Handler will recover panics, reporting
// them to the configured tracer. To override this behaviour, use
// WithRecovery.
func Wrap(h http.Handler, o ...ServerOption) http.Handler {
	if h == nil {
		panic("h == nil")
	}
	handler := &handler{
		handler:        h,
		tracer:         apm.DefaultTracer,
		requestName:    ServerRequestName,
		requestIgnorer: DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(handler)
	}
	if handler.recovery == nil {
		handler.recovery = NewTraceRecovery(handler.tracer)
	}
	return handler
}

// handler wraps an http.Handler, reporting a new transaction for each request.
//
// The http.Request's context will be updated with the transaction.
type handler struct {
	handler          http.Handler
	tracer           *apm.Tracer
	recovery         RecoveryFunc
	panicPropagation bool
	requestName      RequestNameFunc
	requestIgnorer   RequestIgnorerFunc
}

// ServeHTTP delegates to h.Handler, tracing the transaction with
// h.Tracer, or apm.DefaultTracer if h.Tracer is nil.
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !h.tracer.Active() || h.requestIgnorer(req) {
		h.handler.ServeHTTP(w, req)
		return
	}
	tx, req := StartTransaction(h.tracer, h.requestName(req), req)
	defer tx.End()

	body := h.tracer.CaptureHTTPRequestBody(req)
	w, resp := WrapResponseWriter(w)
	defer func() {
		if v := recover(); v != nil {
			if h.panicPropagation {
				defer panic(v)
				// 500 status code will be set only for APM transaction
				// to allow other middleware to choose a different response code
				if resp.StatusCode == 0 {
					resp.StatusCode = http.StatusInternalServerError
				}
			} else if resp.StatusCode == 0 {
				w.WriteHeader(http.StatusInternalServerError)
			}
			h.recovery(w, req, resp, body, tx, v)
		}
		SetTransactionContext(tx, req, resp, body)
		body.Discard()
	}()
	h.handler.ServeHTTP(w, req)
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}
}

// StartTransaction returns a new Transaction with name,
// created with tracer, and taking trace context from req.
//
// If the transaction is not ignored, the request will be
// returned with the transaction added to its context.
func StartTransaction(tracer *apm.Tracer, name string, req *http.Request) (*apm.Transaction, *http.Request) {
	traceContext, ok := getRequestTraceparent(req, ElasticTraceparentHeader)
	if !ok {
		traceContext, ok = getRequestTraceparent(req, W3CTraceparentHeader)
	}
	if ok {
		traceContext.State, _ = ParseTracestateHeader(req.Header[TracestateHeader]...)
	}
	tx := tracer.StartTransactionOptions(name, "request", apm.TransactionOptions{TraceContext: traceContext})
	ctx := apm.ContextWithTransaction(req.Context(), tx)
	req = RequestWithContext(ctx, req)
	return tx, req
}

func getRequestTraceparent(req *http.Request, header string) (apm.TraceContext, bool) {
	if values := req.Header[header]; len(values) == 1 && values[0] != "" {
		if c, err := ParseTraceparentHeader(values[0]); err == nil {
			return c, true
		}
	}
	return apm.TraceContext{}, false
}

// SetTransactionContext sets tx.Result and, if the transaction is being
// sampled, sets tx.Context with information from req, resp, and body.
func SetTransactionContext(tx *apm.Transaction, req *http.Request, resp *Response, body *apm.BodyCapturer) {
	tx.Result = StatusCodeResult(resp.StatusCode)
	if !tx.Sampled() {
		return
	}
	SetContext(&tx.Context, req, resp, body)
}

// SetContext sets the context for a transaction or error using information
// from req, resp, and body.
func SetContext(ctx *apm.Context, req *http.Request, resp *Response, body *apm.BodyCapturer) {
	ctx.SetHTTPRequest(req)
	ctx.SetHTTPRequestBody(body)
	ctx.SetHTTPStatusCode(resp.StatusCode)
	ctx.SetHTTPResponseHeaders(resp.Headers)
}

// WrapResponseWriter wraps an http.ResponseWriter and returns the wrapped
// value along with a *Response which will be filled in when the handler
// is called. The *Response value must not be inspected until after the
// request has been handled, to avoid data races. If neither of the
// ResponseWriter's Write or WriteHeader methods are called, then the
// response's StatusCode field will be zero.
//
// The returned http.ResponseWriter implements http.Pusher and http.Hijacker
// if and only if the provided http.ResponseWriter does.
func WrapResponseWriter(w http.ResponseWriter) (http.ResponseWriter, *Response) {
	rw := responseWriter{
		ResponseWriter: w,
		resp: Response{
			Headers: w.Header(),
		},
	}
	h, _ := w.(http.Hijacker)
	p, _ := w.(http.Pusher)
	switch {
	case h != nil && p != nil:
		rwhp := &responseWriterHijackerPusher{
			responseWriter: rw,
			Hijacker:       h,
			Pusher:         p,
		}
		return rwhp, &rwhp.resp
	case h != nil:
		rwh := &responseWriterHijacker{
			responseWriter: rw,
			Hijacker:       h,
		}
		return rwh, &rwh.resp
	case p != nil:
		rwp := &responseWriterPusher{
			responseWriter: rw,
			Pusher:         p,
		}
		return rwp, &rwp.resp
	}
	return &rw, &rw.resp
}

// Response records details of the HTTP response.
type Response struct {
	// StatusCode records the HTTP status code set via WriteHeader.
	StatusCode int

	// Headers holds the headers set in the ResponseWriter.
	Headers http.Header
}

type responseWriter struct {
	http.ResponseWriter
	resp Response
}

// WriteHeader sets w.resp.StatusCode and calls through to the embedded
// ResponseWriter.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.resp.StatusCode = statusCode
}

// Write calls through to the embedded ResponseWriter, setting
// w.resp.StatusCode to http.StatusOK if WriteHeader has not already
// been called.
func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if w.resp.StatusCode == 0 {
		w.resp.StatusCode = http.StatusOK
	}
	return n, err
}

// CloseNotify returns w.closeNotify() if w.closeNotify is non-nil,
// otherwise it returns nil.
func (w *responseWriter) CloseNotify() <-chan bool {
	if closeNotifier, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return closeNotifier.CloseNotify()
	}
	return nil
}

// Flush calls w.flush() if w.flush is non-nil, otherwise
// it does nothing.
func (w *responseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type responseWriterHijacker struct {
	responseWriter
	http.Hijacker
}

type responseWriterPusher struct {
	responseWriter
	http.Pusher
}

type responseWriterHijackerPusher struct {
	responseWriter
	http.Hijacker
	http.Pusher
}

// ServerOption sets options for tracing server requests.
type ServerOption func(*handler)

// WithTracer returns a ServerOption which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) ServerOption {
	if t == nil {
		panic("t == nil")
	}
	return func(h *handler) {
		h.tracer = t
	}
}

// WithRecovery returns a ServerOption which sets r as the recovery
// function to use for tracing server requests.
func WithRecovery(r RecoveryFunc) ServerOption {
	if r == nil {
		panic("r == nil")
	}
	return func(h *handler) {
		h.recovery = r
	}
}

// WithPanicPropagation returns a ServerOption which enable panic propagation.
// Any panic will be recovered and recorded as an error in a transaction, then
// panic will be caused again.
func WithPanicPropagation() ServerOption {
	return func(h *handler) {
		h.panicPropagation = true
	}
}

// RequestNameFunc is the type of a function for use in
// WithServerRequestName.
type RequestNameFunc func(*http.Request) string

// WithServerRequestName returns a ServerOption which sets r as the function
// to use to obtain the transaction name for the given server request.
func WithServerRequestName(r RequestNameFunc) ServerOption {
	if r == nil {
		panic("r == nil")
	}
	return func(h *handler) {
		h.requestName = r
	}
}

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc func(*http.Request) bool

// WithServerRequestIgnorer returns a ServerOption which sets r as the
// function to use to determine whether or not a server request should
// be ignored. If r is nil, all requests will be reported.
func WithServerRequestIgnorer(r RequestIgnorerFunc) ServerOption {
	if r == nil {
		r = IgnoreNone
	}
	return func(h *handler) {
		h.requestIgnorer = r
	}
}

// RequestWithContext is equivalent to req.WithContext, except that the URL
// pointer is copied, rather than the contents.
func RequestWithContext(ctx context.Context, req *http.Request) *http.Request {
	url := req.URL
	req.URL = nil
	reqCopy := req.WithContext(ctx)
	reqCopy.URL = url
	req.URL = url
	return reqCopy
}
