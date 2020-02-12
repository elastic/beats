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

func newTransformPutTransformFunc(t Transport) TransformPutTransform {
	return func(body io.Reader, transform_id string, o ...func(*TransformPutTransformRequest)) (*Response, error) {
		var r = TransformPutTransformRequest{Body: body, TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformPutTransform -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/put-transform.html.
//
type TransformPutTransform func(body io.Reader, transform_id string, o ...func(*TransformPutTransformRequest)) (*Response, error)

// TransformPutTransformRequest configures the Transform Put Transform API request.
//
type TransformPutTransformRequest struct {
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
func (r TransformPutTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID))
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)

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
func (f TransformPutTransform) WithContext(v context.Context) func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.ctx = v
	}
}

// WithDeferValidation - if validations should be deferred until transform starts, defaults to false..
//
func (f TransformPutTransform) WithDeferValidation(v bool) func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.DeferValidation = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformPutTransform) WithPretty() func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformPutTransform) WithHuman() func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformPutTransform) WithErrorTrace() func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformPutTransform) WithFilterPath(v ...string) func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformPutTransform) WithHeader(h map[string]string) func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
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
func (f TransformPutTransform) WithOpaqueID(s string) func(*TransformPutTransformRequest) {
	return func(r *TransformPutTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
