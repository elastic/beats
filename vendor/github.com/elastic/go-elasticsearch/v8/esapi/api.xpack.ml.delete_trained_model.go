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

func newMLDeleteTrainedModelFunc(t Transport) MLDeleteTrainedModel {
	return func(model_id string, o ...func(*MLDeleteTrainedModelRequest)) (*Response, error) {
		var r = MLDeleteTrainedModelRequest{ModelID: model_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLDeleteTrainedModel -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-inference.html.
//
type MLDeleteTrainedModel func(model_id string, o ...func(*MLDeleteTrainedModelRequest)) (*Response, error)

// MLDeleteTrainedModelRequest configures the ML Delete Trained Model API request.
//
type MLDeleteTrainedModelRequest struct {
	ModelID string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLDeleteTrainedModelRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_ml") + 1 + len("inference") + 1 + len(r.ModelID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("inference")
	path.WriteString("/")
	path.WriteString(r.ModelID)

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
func (f MLDeleteTrainedModel) WithContext(v context.Context) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLDeleteTrainedModel) WithPretty() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLDeleteTrainedModel) WithHuman() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLDeleteTrainedModel) WithErrorTrace() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLDeleteTrainedModel) WithFilterPath(v ...string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLDeleteTrainedModel) WithHeader(h map[string]string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
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
func (f MLDeleteTrainedModel) WithOpaqueID(s string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
