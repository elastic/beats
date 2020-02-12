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
	"time"
)

func newTransformStartTransformFunc(t Transport) TransformStartTransform {
	return func(transform_id string, o ...func(*TransformStartTransformRequest)) (*Response, error) {
		var r = TransformStartTransformRequest{TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformStartTransform -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/start-transform.html.
//
type TransformStartTransform func(transform_id string, o ...func(*TransformStartTransformRequest)) (*Response, error)

// TransformStartTransformRequest configures the Transform Start Transform API request.
//
type TransformStartTransformRequest struct {
	TransformID string

	Timeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r TransformStartTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID) + 1 + len("_start"))
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)
	path.WriteString("/")
	path.WriteString("_start")

	params = make(map[string]string)

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

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
func (f TransformStartTransform) WithContext(v context.Context) func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.ctx = v
	}
}

// WithTimeout - controls the time to wait for the transform to start.
//
func (f TransformStartTransform) WithTimeout(v time.Duration) func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformStartTransform) WithPretty() func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformStartTransform) WithHuman() func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformStartTransform) WithErrorTrace() func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformStartTransform) WithFilterPath(v ...string) func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformStartTransform) WithHeader(h map[string]string) func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
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
func (f TransformStartTransform) WithOpaqueID(s string) func(*TransformStartTransformRequest) {
	return func(r *TransformStartTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
