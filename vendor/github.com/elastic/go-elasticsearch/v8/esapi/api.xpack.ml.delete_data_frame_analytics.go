// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 8.0.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

func newMLDeleteDataFrameAnalyticsFunc(t Transport) MLDeleteDataFrameAnalytics {
	return func(id string, o ...func(*MLDeleteDataFrameAnalyticsRequest)) (*Response, error) {
		var r = MLDeleteDataFrameAnalyticsRequest{ID: id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLDeleteDataFrameAnalytics -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-dfanalytics.html.
//
type MLDeleteDataFrameAnalytics func(id string, o ...func(*MLDeleteDataFrameAnalyticsRequest)) (*Response, error)

// MLDeleteDataFrameAnalyticsRequest configures the ML Delete Data Frame Analytics API request.
//
type MLDeleteDataFrameAnalyticsRequest struct {
	ID string

	Force *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLDeleteDataFrameAnalyticsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.ID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	path.WriteString("/")
	path.WriteString(r.ID)

	params = make(map[string]string)

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
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
func (f MLDeleteDataFrameAnalytics) WithContext(v context.Context) func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.ctx = v
	}
}

// WithForce - true if the job should be forcefully deleted.
//
func (f MLDeleteDataFrameAnalytics) WithForce(v bool) func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.Force = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLDeleteDataFrameAnalytics) WithPretty() func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLDeleteDataFrameAnalytics) WithHuman() func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLDeleteDataFrameAnalytics) WithErrorTrace() func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLDeleteDataFrameAnalytics) WithFilterPath(v ...string) func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLDeleteDataFrameAnalytics) WithHeader(h map[string]string) func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
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
func (f MLDeleteDataFrameAnalytics) WithOpaqueID(s string) func(*MLDeleteDataFrameAnalyticsRequest) {
	return func(r *MLDeleteDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
