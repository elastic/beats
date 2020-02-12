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

func newEnrichDeletePolicyFunc(t Transport) EnrichDeletePolicy {
	return func(name string, o ...func(*EnrichDeletePolicyRequest)) (*Response, error) {
		var r = EnrichDeletePolicyRequest{Name: name}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// EnrichDeletePolicy -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-enrich-policy-api.html.
//
type EnrichDeletePolicy func(name string, o ...func(*EnrichDeletePolicyRequest)) (*Response, error)

// EnrichDeletePolicyRequest configures the Enrich Delete Policy API request.
//
type EnrichDeletePolicyRequest struct {
	Name string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r EnrichDeletePolicyRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_enrich") + 1 + len("policy") + 1 + len(r.Name))
	path.WriteString("/")
	path.WriteString("_enrich")
	path.WriteString("/")
	path.WriteString("policy")
	path.WriteString("/")
	path.WriteString(r.Name)

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
func (f EnrichDeletePolicy) WithContext(v context.Context) func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f EnrichDeletePolicy) WithPretty() func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f EnrichDeletePolicy) WithHuman() func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f EnrichDeletePolicy) WithErrorTrace() func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f EnrichDeletePolicy) WithFilterPath(v ...string) func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f EnrichDeletePolicy) WithHeader(h map[string]string) func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
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
func (f EnrichDeletePolicy) WithOpaqueID(s string) func(*EnrichDeletePolicyRequest) {
	return func(r *EnrichDeletePolicyRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
