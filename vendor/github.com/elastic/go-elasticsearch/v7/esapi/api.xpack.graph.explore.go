// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 7.5.0: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

func newGraphExploreFunc(t Transport) GraphExplore {
	return func(index []string, o ...func(*GraphExploreRequest)) (*Response, error) {
		var r = GraphExploreRequest{Index: index}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// GraphExplore -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/graph-explore-api.html.
//
type GraphExplore func(index []string, o ...func(*GraphExploreRequest)) (*Response, error)

// GraphExploreRequest configures the Graph Explore API request.
//
type GraphExploreRequest struct {
	Index        []string
	DocumentType []string

	Body io.Reader

	Routing string
	Timeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r GraphExploreRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len(strings.Join(r.DocumentType, ",")) + 1 + len("_graph") + 1 + len("explore"))
	path.WriteString("/")
	path.WriteString(strings.Join(r.Index, ","))
	if len(r.DocumentType) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.DocumentType, ","))
	}
	path.WriteString("/")
	path.WriteString("_graph")
	path.WriteString("/")
	path.WriteString("explore")

	params = make(map[string]string)

	if r.Routing != "" {
		params["routing"] = r.Routing
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
func (f GraphExplore) WithContext(v context.Context) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.ctx = v
	}
}

// WithBody - Graph Query DSL.
//
func (f GraphExplore) WithBody(v io.Reader) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.Body = v
	}
}

// WithDocumentType - a list of document types to search; leave empty to perform the operation on all types.
//
func (f GraphExplore) WithDocumentType(v ...string) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.DocumentType = v
	}
}

// WithRouting - specific routing value.
//
func (f GraphExplore) WithRouting(v string) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.Routing = v
	}
}

// WithTimeout - explicit operation timeout.
//
func (f GraphExplore) WithTimeout(v time.Duration) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f GraphExplore) WithPretty() func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f GraphExplore) WithHuman() func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f GraphExplore) WithErrorTrace() func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f GraphExplore) WithFilterPath(v ...string) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f GraphExplore) WithHeader(h map[string]string) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
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
func (f GraphExplore) WithOpaqueID(s string) func(*GraphExploreRequest) {
	return func(r *GraphExploreRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
