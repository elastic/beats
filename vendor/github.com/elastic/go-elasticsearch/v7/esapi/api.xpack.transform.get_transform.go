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

func newTransformGetTransformFunc(t Transport) TransformGetTransform {
	return func(o ...func(*TransformGetTransformRequest)) (*Response, error) {
		var r = TransformGetTransformRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformGetTransform -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-transform.html.
//
type TransformGetTransform func(o ...func(*TransformGetTransformRequest)) (*Response, error)

// TransformGetTransformRequest configures the Transform Get Transform API request.
//
type TransformGetTransformRequest struct {
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
func (r TransformGetTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID))
	path.WriteString("/")
	path.WriteString("_transform")
	if r.TransformID != "" {
		path.WriteString("/")
		path.WriteString(r.TransformID)
	}

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
func (f TransformGetTransform) WithContext(v context.Context) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.ctx = v
	}
}

// WithTransformID - the ID or comma delimited list of ID expressions of the transforms to get, '_all' or '*' implies get all transforms.
//
func (f TransformGetTransform) WithTransformID(v string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.TransformID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no transforms. (this includes `_all` string or when no transforms have been specified).
//
func (f TransformGetTransform) WithAllowNoMatch(v bool) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFrom - skips a number of transform configs, defaults to 0.
//
func (f TransformGetTransform) WithFrom(v int) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of transforms to get, defaults to 100.
//
func (f TransformGetTransform) WithSize(v int) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformGetTransform) WithPretty() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformGetTransform) WithHuman() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformGetTransform) WithErrorTrace() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformGetTransform) WithFilterPath(v ...string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformGetTransform) WithHeader(h map[string]string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
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
func (f TransformGetTransform) WithOpaqueID(s string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
