// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strings"
	"time"
)

func newIndicesUpdateAliasesFunc(t Transport) IndicesUpdateAliases {
	return func(body io.Reader, o ...func(*IndicesUpdateAliasesRequest)) (*Response, error) {
		var r = IndicesUpdateAliasesRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// IndicesUpdateAliases updates index aliases.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/indices-aliases.html.
//
type IndicesUpdateAliases func(body io.Reader, o ...func(*IndicesUpdateAliasesRequest)) (*Response, error)

// IndicesUpdateAliasesRequest configures the Indices  Update Aliases API request.
//
type IndicesUpdateAliasesRequest struct {
	Body io.Reader

	MasterTimeout time.Duration
	Timeout       time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r IndicesUpdateAliasesRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_aliases"))
	path.WriteString("/_aliases")

	params = make(map[string]string)

	if r.MasterTimeout != 0 {
		params["master_timeout"] = time.Duration(r.MasterTimeout * time.Millisecond).String()
	}

	if r.Timeout != 0 {
		params["timeout"] = time.Duration(r.Timeout * time.Millisecond).String()
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
func (f IndicesUpdateAliases) WithContext(v context.Context) func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - specify timeout for connection to master.
//
func (f IndicesUpdateAliases) WithMasterTimeout(v time.Duration) func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - request timeout.
//
func (f IndicesUpdateAliases) WithTimeout(v time.Duration) func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f IndicesUpdateAliases) WithPretty() func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f IndicesUpdateAliases) WithHuman() func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f IndicesUpdateAliases) WithErrorTrace() func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f IndicesUpdateAliases) WithFilterPath(v ...string) func(*IndicesUpdateAliasesRequest) {
	return func(r *IndicesUpdateAliasesRequest) {
		r.FilterPath = v
	}
}
