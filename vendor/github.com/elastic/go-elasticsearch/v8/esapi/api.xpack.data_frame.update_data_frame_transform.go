// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package esapi

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newDataFrameUpdateDataFrameTransformFunc(t Transport) DataFrameUpdateDataFrameTransform {
	return func(body io.Reader, transform_id string, o ...func(*DataFrameUpdateDataFrameTransformRequest)) (*Response, error) {
		var r = DataFrameUpdateDataFrameTransformRequest{Body: body, TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// DataFrameUpdateDataFrameTransform - https://www.elastic.co/guide/en/elasticsearch/reference/current/update-data-frame-transform.html
//
type DataFrameUpdateDataFrameTransform func(body io.Reader, transform_id string, o ...func(*DataFrameUpdateDataFrameTransformRequest)) (*Response, error)

// DataFrameUpdateDataFrameTransformRequest configures the Data Frame Update Data Frame Transform API request.
//
type DataFrameUpdateDataFrameTransformRequest struct {
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
func (r DataFrameUpdateDataFrameTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_data_frame") + 1 + len("transforms") + 1 + len(r.TransformID) + 1 + len("_update"))
	path.WriteString("/")
	path.WriteString("_data_frame")
	path.WriteString("/")
	path.WriteString("transforms")
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

	req, _ := newRequest(method, path.String(), r.Body)

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
func (f DataFrameUpdateDataFrameTransform) WithContext(v context.Context) func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.ctx = v
	}
}

// WithDeferValidation - if validations should be deferred until data frame transform starts, defaults to false..
//
func (f DataFrameUpdateDataFrameTransform) WithDeferValidation(v bool) func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.DeferValidation = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f DataFrameUpdateDataFrameTransform) WithPretty() func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f DataFrameUpdateDataFrameTransform) WithHuman() func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f DataFrameUpdateDataFrameTransform) WithErrorTrace() func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f DataFrameUpdateDataFrameTransform) WithFilterPath(v ...string) func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f DataFrameUpdateDataFrameTransform) WithHeader(h map[string]string) func(*DataFrameUpdateDataFrameTransformRequest) {
	return func(r *DataFrameUpdateDataFrameTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}
