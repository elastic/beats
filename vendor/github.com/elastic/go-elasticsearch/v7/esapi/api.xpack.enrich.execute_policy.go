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

func newEnrichExecutePolicyFunc(t Transport) EnrichExecutePolicy {
	return func(name string, o ...func(*EnrichExecutePolicyRequest)) (*Response, error) {
		var r = EnrichExecutePolicyRequest{Name: name}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// EnrichExecutePolicy -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/enrich-execute-policy.html.
//
type EnrichExecutePolicy func(name string, o ...func(*EnrichExecutePolicyRequest)) (*Response, error)

// EnrichExecutePolicyRequest configures the Enrich Execute Policy API request.
//
type EnrichExecutePolicyRequest struct {
	Name string

	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r EnrichExecutePolicyRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_enrich") + 1 + len("policy") + 1 + len(r.Name) + 1 + len("_execute"))
	path.WriteString("/")
	path.WriteString("_enrich")
	path.WriteString("/")
	path.WriteString("policy")
	path.WriteString("/")
	path.WriteString(r.Name)
	path.WriteString("/")
	path.WriteString("_execute")

	params = make(map[string]string)

	if r.WaitForCompletion != nil {
		params["wait_for_completion"] = strconv.FormatBool(*r.WaitForCompletion)
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
func (f EnrichExecutePolicy) WithContext(v context.Context) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.ctx = v
	}
}

// WithWaitForCompletion - should the request should block until the execution is complete..
//
func (f EnrichExecutePolicy) WithWaitForCompletion(v bool) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f EnrichExecutePolicy) WithPretty() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f EnrichExecutePolicy) WithHuman() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f EnrichExecutePolicy) WithErrorTrace() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f EnrichExecutePolicy) WithFilterPath(v ...string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f EnrichExecutePolicy) WithHeader(h map[string]string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
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
func (f EnrichExecutePolicy) WithOpaqueID(s string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
