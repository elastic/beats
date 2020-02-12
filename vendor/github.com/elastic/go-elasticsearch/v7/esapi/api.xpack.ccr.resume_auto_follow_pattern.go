// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 7.5.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newCCRResumeAutoFollowPatternFunc(t Transport) CCRResumeAutoFollowPattern {
	return func(name string, o ...func(*CCRResumeAutoFollowPatternRequest)) (*Response, error) {
		var r = CCRResumeAutoFollowPatternRequest{Name: name}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// CCRResumeAutoFollowPattern -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ccr-resume-auto-follow-pattern.html.
//
type CCRResumeAutoFollowPattern func(name string, o ...func(*CCRResumeAutoFollowPatternRequest)) (*Response, error)

// CCRResumeAutoFollowPatternRequest configures the CCR Resume Auto Follow Pattern API request.
//
type CCRResumeAutoFollowPatternRequest struct {
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
func (r CCRResumeAutoFollowPatternRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_ccr") + 1 + len("auto_follow") + 1 + len(r.Name) + 1 + len("resume"))
	path.WriteString("/")
	path.WriteString("_ccr")
	path.WriteString("/")
	path.WriteString("auto_follow")
	path.WriteString("/")
	path.WriteString(r.Name)
	path.WriteString("/")
	path.WriteString("resume")

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
func (f CCRResumeAutoFollowPattern) WithContext(v context.Context) func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f CCRResumeAutoFollowPattern) WithPretty() func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f CCRResumeAutoFollowPattern) WithHuman() func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f CCRResumeAutoFollowPattern) WithErrorTrace() func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f CCRResumeAutoFollowPattern) WithFilterPath(v ...string) func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f CCRResumeAutoFollowPattern) WithHeader(h map[string]string) func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
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
func (f CCRResumeAutoFollowPattern) WithOpaqueID(s string) func(*CCRResumeAutoFollowPatternRequest) {
	return func(r *CCRResumeAutoFollowPatternRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
