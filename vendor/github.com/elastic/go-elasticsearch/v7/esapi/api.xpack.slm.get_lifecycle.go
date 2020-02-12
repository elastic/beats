// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 7.5.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newSlmGetLifecycleFunc(t Transport) SlmGetLifecycle {
	return func(o ...func(*SlmGetLifecycleRequest)) (*Response, error) {
		var r = SlmGetLifecycleRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SlmGetLifecycle -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/slm-api-get.html.
//
type SlmGetLifecycle func(o ...func(*SlmGetLifecycleRequest)) (*Response, error)

// SlmGetLifecycleRequest configures the Slm Get Lifecycle API request.
//
type SlmGetLifecycleRequest struct {
	PolicyID []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SlmGetLifecycleRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_slm") + 1 + len("policy") + 1 + len(strings.Join(r.PolicyID, ",")))
	path.WriteString("/")
	path.WriteString("_slm")
	path.WriteString("/")
	path.WriteString("policy")
	if len(r.PolicyID) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.PolicyID, ","))
	}

	params = make(map[string]string)

	if r.Pretty {
		params["pretty"] = "true"
	}

	if r.Human {
		params["human"] = "true"
	}

	if r.ErrorTrace {
		params["error_trace"] = "true"
	}

	if len(r.FilterPath) > 0 {
		params["filter_path"] = strings.Join(r.FilterPath, ",")
	}

	req, err := newRequest(method, path.String(), nil)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	res, err := transport.Perform(req)
	if err != nil {
		return nil, err
	}

	response := Response{
		StatusCode: res.StatusCode,
		Body:       res.Body,
		Header:     res.Header,
	}

	return &response, nil
}

// WithContext sets the request context.
//
func (f SlmGetLifecycle) WithContext(v context.Context) func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.ctx = v
	}
}

// WithPolicyID - comma-separated list of snapshot lifecycle policies to retrieve.
//
func (f SlmGetLifecycle) WithPolicyID(v ...string) func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.PolicyID = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SlmGetLifecycle) WithPretty() func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SlmGetLifecycle) WithHuman() func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SlmGetLifecycle) WithErrorTrace() func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SlmGetLifecycle) WithFilterPath(v ...string) func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SlmGetLifecycle) WithHeader(h map[string]string) func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
//
func (f SlmGetLifecycle) WithOpaqueID(s string) func(*SlmGetLifecycleRequest) {
	return func(r *SlmGetLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
