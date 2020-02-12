// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 7.5.0: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func newMLEstimateMemoryUsageFunc(t Transport) MLEstimateMemoryUsage {
	return func(body io.Reader, o ...func(*MLEstimateMemoryUsageRequest)) (*Response, error) {
		var r = MLEstimateMemoryUsageRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLEstimateMemoryUsage -
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/estimate-memory-usage-dfanalytics.html.
//
type MLEstimateMemoryUsage func(body io.Reader, o ...func(*MLEstimateMemoryUsageRequest)) (*Response, error)

// MLEstimateMemoryUsageRequest configures the ML Estimate Memory Usage API request.
//
type MLEstimateMemoryUsageRequest struct {
	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLEstimateMemoryUsageRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_ml/data_frame/analytics/_estimate_memory_usage"))
	path.WriteString("/_ml/data_frame/analytics/_estimate_memory_usage")

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

	req, err := newRequest(method, path.String(), r.Body)
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

	if r.Body != nil {
		req.Header[headerContentType] = headerContentTypeJSON
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
func (f MLEstimateMemoryUsage) WithContext(v context.Context) func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLEstimateMemoryUsage) WithPretty() func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLEstimateMemoryUsage) WithHuman() func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLEstimateMemoryUsage) WithErrorTrace() func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLEstimateMemoryUsage) WithFilterPath(v ...string) func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLEstimateMemoryUsage) WithHeader(h map[string]string) func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
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
func (f MLEstimateMemoryUsage) WithOpaqueID(s string) func(*MLEstimateMemoryUsageRequest) {
	return func(r *MLEstimateMemoryUsageRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
