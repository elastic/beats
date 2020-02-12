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
	"strconv"
	"strings"
)

func newTransformUpdateTransformFunc(t Transport) TransformUpdateTransform {
	return func(body io.Reader, transform_id string, o ...func(*TransformUpdateTransformRequest)) (*Response, error) {
		var r = TransformUpdateTransformRequest{Body: body, TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformUpdateTransform -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/update-transform.html.
//
type TransformUpdateTransform func(body io.Reader, transform_id string, o ...func(*TransformUpdateTransformRequest)) (*Response, error)

// TransformUpdateTransformRequest configures the Transform Update Transform API request.
//
type TransformUpdateTransformRequest struct {
	Body io.Reader

	TransformID string

	DeferValidation *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r TransformUpdateTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID) + 1 + len("_update"))
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)
	path.WriteString("/")
	path.WriteString("_update")

	params = make(map[string]string)

	if r.DeferValidation != nil {
		params["defer_validation"] = strconv.FormatBool(*r.DeferValidation)
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
func (f TransformUpdateTransform) WithContext(v context.Context) func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.ctx = v
	}
}

// WithDeferValidation - if validations should be deferred until transform starts, defaults to false..
//
func (f TransformUpdateTransform) WithDeferValidation(v bool) func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.DeferValidation = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformUpdateTransform) WithPretty() func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformUpdateTransform) WithHuman() func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformUpdateTransform) WithErrorTrace() func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformUpdateTransform) WithFilterPath(v ...string) func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformUpdateTransform) WithHeader(h map[string]string) func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
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
func (f TransformUpdateTransform) WithOpaqueID(s string) func(*TransformUpdateTransformRequest) {
	return func(r *TransformUpdateTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
