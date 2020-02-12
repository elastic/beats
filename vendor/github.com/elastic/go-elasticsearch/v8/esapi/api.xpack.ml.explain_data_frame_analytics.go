// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 8.0.0: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func newMLExplainDataFrameAnalyticsFunc(t Transport) MLExplainDataFrameAnalytics {
	return func(o ...func(*MLExplainDataFrameAnalyticsRequest)) (*Response, error) {
		var r = MLExplainDataFrameAnalyticsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLExplainDataFrameAnalytics -
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/explain-dfanalytics.html.
//
type MLExplainDataFrameAnalytics func(o ...func(*MLExplainDataFrameAnalyticsRequest)) (*Response, error)

// MLExplainDataFrameAnalyticsRequest configures the ML Explain Data Frame Analytics API request.
//
type MLExplainDataFrameAnalyticsRequest struct {
	DocumentID string

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
func (r MLExplainDataFrameAnalyticsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.DocumentID) + 1 + len("_explain"))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
	}
	path.WriteString("/")
	path.WriteString("_explain")

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
func (f MLExplainDataFrameAnalytics) WithContext(v context.Context) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.ctx = v
	}
}

// WithBody - The data frame analytics config to explain.
//
func (f MLExplainDataFrameAnalytics) WithBody(v io.Reader) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Body = v
	}
}

// WithDocumentID - the ID of the data frame analytics to explain.
//
func (f MLExplainDataFrameAnalytics) WithDocumentID(v string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.DocumentID = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLExplainDataFrameAnalytics) WithPretty() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLExplainDataFrameAnalytics) WithHuman() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLExplainDataFrameAnalytics) WithErrorTrace() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLExplainDataFrameAnalytics) WithFilterPath(v ...string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLExplainDataFrameAnalytics) WithHeader(h map[string]string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
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
func (f MLExplainDataFrameAnalytics) WithOpaqueID(s string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
