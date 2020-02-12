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

func newGetScriptLanguagesFunc(t Transport) GetScriptLanguages {
	return func(o ...func(*GetScriptLanguagesRequest)) (*Response, error) {
		var r = GetScriptLanguagesRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// GetScriptLanguages returns available script types, languages and contexts
//
type GetScriptLanguages func(o ...func(*GetScriptLanguagesRequest)) (*Response, error)

// GetScriptLanguagesRequest configures the Get Script Languages API request.
//
type GetScriptLanguagesRequest struct {
	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r GetScriptLanguagesRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(len("/_script_language"))
	path.WriteString("/_script_language")

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
func (f GetScriptLanguages) WithContext(v context.Context) func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f GetScriptLanguages) WithPretty() func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f GetScriptLanguages) WithHuman() func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f GetScriptLanguages) WithErrorTrace() func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f GetScriptLanguages) WithFilterPath(v ...string) func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f GetScriptLanguages) WithHeader(h map[string]string) func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
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
func (f GetScriptLanguages) WithOpaqueID(s string) func(*GetScriptLanguagesRequest) {
	return func(r *GetScriptLanguagesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
