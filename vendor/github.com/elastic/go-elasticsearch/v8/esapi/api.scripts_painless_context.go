// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package esapi

import (
	"context"
	"strings"
)

func newScriptsPainlessContextFunc(t Transport) ScriptsPainlessContext {
	return func(o ...func(*ScriptsPainlessContextRequest)) (*Response, error) {
		var r = ScriptsPainlessContextRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// ScriptsPainlessContext allows to query context information.
//
type ScriptsPainlessContext func(o ...func(*ScriptsPainlessContextRequest)) (*Response, error)

// ScriptsPainlessContextRequest configures the Scripts Painless Context API request.
//
type ScriptsPainlessContextRequest struct {
	ScriptContext string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r ScriptsPainlessContextRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(len("/_scripts/painless/_context"))
	path.WriteString("/_scripts/painless/_context")

	params = make(map[string]string)

	if r.ScriptContext != "" {
		params["context"] = r.ScriptContext
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

	req, _ := newRequest(method, path.String(), nil)

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
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
func (f ScriptsPainlessContext) WithContext(v context.Context) func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.ctx = v
	}
}

// WithScriptContext - select a specific context to retrieve api information about.
//
func (f ScriptsPainlessContext) WithScriptContext(v string) func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.ScriptContext = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f ScriptsPainlessContext) WithPretty() func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f ScriptsPainlessContext) WithHuman() func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f ScriptsPainlessContext) WithErrorTrace() func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f ScriptsPainlessContext) WithFilterPath(v ...string) func(*ScriptsPainlessContextRequest) {
	return func(r *ScriptsPainlessContextRequest) {
		r.FilterPath = v
	}
}
