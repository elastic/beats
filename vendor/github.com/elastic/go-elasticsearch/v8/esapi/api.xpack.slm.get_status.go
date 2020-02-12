// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 8.0.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newSlmGetStatusFunc(t Transport) SlmGetStatus {
	return func(o ...func(*SlmGetStatusRequest)) (*Response, error) {
		var r = SlmGetStatusRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SlmGetStatus -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/slm-api-get-status.html.
//
type SlmGetStatus func(o ...func(*SlmGetStatusRequest)) (*Response, error)

// SlmGetStatusRequest configures the Slm Get Status API request.
//
type SlmGetStatusRequest struct {
	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SlmGetStatusRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(len("/_slm/status"))
	path.WriteString("/_slm/status")

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
func (f SlmGetStatus) WithContext(v context.Context) func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SlmGetStatus) WithPretty() func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SlmGetStatus) WithHuman() func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SlmGetStatus) WithErrorTrace() func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SlmGetStatus) WithFilterPath(v ...string) func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SlmGetStatus) WithHeader(h map[string]string) func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
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
func (f SlmGetStatus) WithOpaqueID(s string) func(*SlmGetStatusRequest) {
	return func(r *SlmGetStatusRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
