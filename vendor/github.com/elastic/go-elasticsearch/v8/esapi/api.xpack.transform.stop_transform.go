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
	"time"
)

func newTransformStopTransformFunc(t Transport) TransformStopTransform {
	return func(transform_id string, o ...func(*TransformStopTransformRequest)) (*Response, error) {
		var r = TransformStopTransformRequest{TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TransformStopTransform -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/stop-transform.html.
//
type TransformStopTransform func(transform_id string, o ...func(*TransformStopTransformRequest)) (*Response, error)

// TransformStopTransformRequest configures the Transform Stop Transform API request.
//
type TransformStopTransformRequest struct {
	TransformID string

	AllowNoMatch      *bool
	Force             *bool
	Timeout           time.Duration
	WaitForCheckpoint *bool
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
func (r TransformStopTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_transform") + 1 + len(r.TransformID) + 1 + len("_stop"))
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)
	path.WriteString("/")
	path.WriteString("_stop")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.WaitForCheckpoint != nil {
		params["wait_for_checkpoint"] = strconv.FormatBool(*r.WaitForCheckpoint)
	}

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
func (f TransformStopTransform) WithContext(v context.Context) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.ctx = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no transforms. (this includes `_all` string or when no transforms have been specified).
//
func (f TransformStopTransform) WithAllowNoMatch(v bool) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.AllowNoMatch = &v
	}
}

// WithForce - whether to force stop a failed transform or not. default to false.
//
func (f TransformStopTransform) WithForce(v bool) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.Force = &v
	}
}

// WithTimeout - controls the time to wait until the transform has stopped. default to 30 seconds.
//
func (f TransformStopTransform) WithTimeout(v time.Duration) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.Timeout = v
	}
}

// WithWaitForCheckpoint - whether to wait for the transform to reach a checkpoint before stopping. default to false.
//
func (f TransformStopTransform) WithWaitForCheckpoint(v bool) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.WaitForCheckpoint = &v
	}
}

// WithWaitForCompletion - whether to wait for the transform to fully stop before returning or not. default to false.
//
func (f TransformStopTransform) WithWaitForCompletion(v bool) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f TransformStopTransform) WithPretty() func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f TransformStopTransform) WithHuman() func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f TransformStopTransform) WithErrorTrace() func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f TransformStopTransform) WithFilterPath(v ...string) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f TransformStopTransform) WithHeader(h map[string]string) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
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
func (f TransformStopTransform) WithOpaqueID(s string) func(*TransformStopTransformRequest) {
	return func(r *TransformStopTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
