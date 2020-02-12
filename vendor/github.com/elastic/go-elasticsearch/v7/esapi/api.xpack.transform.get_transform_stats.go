// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 7.5.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

func newTransformGetTransformStatsFunc(t Transport) TransformGetTransformStats {
	return func(transform_id string, o ...func(*TransformGetTransformStatsRequest)) (*Response, error) {
		var r = TransformGetTransformStatsRequest{TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformGetTransformStats -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-transform-stats.html.
//
type TransformGetTransformStats func(transform_id string, o ...func(*TransformGetTransformStatsRequest)) (*Response, error)

// TransformGetTransformStatsRequest configures the Transform Get Transform Stats API request.
//
type TransformGetTransformStatsRequest struct {
	TransformID string

	AllowNoMatch *bool
	From         *int
	Size         *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r TransformGetTransformStatsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID) + 1 + len("_stats"))
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)
	path.WriteString("/")
	path.WriteString("_stats")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
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
func (f TransformGetTransformStats) WithContext(v context.Context) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.ctx = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no transforms. (this includes `_all` string or when no transforms have been specified).
//
func (f TransformGetTransformStats) WithAllowNoMatch(v bool) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFrom - skips a number of transform stats, defaults to 0.
//
func (f TransformGetTransformStats) WithFrom(v int) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of transform stats to get, defaults to 100.
//
func (f TransformGetTransformStats) WithSize(v int) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformGetTransformStats) WithPretty() func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformGetTransformStats) WithHuman() func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformGetTransformStats) WithErrorTrace() func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformGetTransformStats) WithFilterPath(v ...string) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformGetTransformStats) WithHeader(h map[string]string) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
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
func (f TransformGetTransformStats) WithOpaqueID(s string) func(*TransformGetTransformStatsRequest) {
	return func(r *TransformGetTransformStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
