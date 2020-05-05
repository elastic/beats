// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmelasticsearch

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync/atomic"
	"unsafe"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// WrapRoundTripper returns an http.RoundTripper wrapping r, reporting each
// request as a span to Elastic APM, if the request's context contains a
// sampled transaction.
//
// If r is nil, then http.DefaultTransport is wrapped.
func WrapRoundTripper(r http.RoundTripper, o ...ClientOption) http.RoundTripper {
	if r == nil {
		r = http.DefaultTransport
	}
	rt := &roundTripper{r: r}
	for _, o := range o {
		o(rt)
	}
	return rt
}

type roundTripper struct {
	r http.RoundTripper
}

// RoundTrip delegates to r.r, emitting a span if req's context contains a transaction.
//
// If req.URL.Path corresponds to a search request, then RoundTrip will attempt to extract
// the search query to use as the span context's "database statement". If the query is
// passed in as a query parameter (i.e. "/_search?q=foo:bar"), then that will be used;
// otherwise, the request body will be read. In the latter case, req.GetBody is used
// if defined, otherwise we read req.Body, preserving its contents for the underlying
// RoundTripper. If the request body is gzip-encoded, it will be decoded.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil || !tx.Sampled() {
		return r.r.RoundTrip(req)
	}

	name := requestName(req)
	span := tx.StartSpan(name, "db.elasticsearch", apm.SpanFromContext(ctx))
	if span.Dropped() {
		span.End()
		return r.r.RoundTrip(req)
	}

	statement, req := captureSearchStatement(req)
	username, _, _ := req.BasicAuth()
	ctx = apm.ContextWithSpan(ctx, span)
	req = apmhttp.RequestWithContext(ctx, req)
	span.Context.SetHTTPRequest(req)
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     "elasticsearch",
		Resource: "elasticsearch",
	})
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: statement,
		User:      username,
	})

	resp, err := r.r.RoundTrip(req)
	if err != nil {
		span.End()
	} else {
		span.Context.SetHTTPStatusCode(resp.StatusCode)
		resp.Body = &responseBody{span: span, body: resp.Body}
	}
	return resp, err
}

type responseBody struct {
	span *apm.Span
	body io.ReadCloser
}

// Close closes the response body, and ends the span if it hasn't already been ended.
func (b *responseBody) Close() error {
	b.endSpan()
	return b.body.Close()
}

// Read reads from the response body, and ends the span when io.EOF is returend if
// the span hasn't already been ended.
func (b *responseBody) Read(p []byte) (n int, err error) {
	n, err = b.body.Read(p)
	if err == io.EOF {
		b.endSpan()
	}
	return n, err
}

func (b *responseBody) endSpan() {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&b.span))
	if old := atomic.SwapPointer(addr, nil); old != nil {
		(*apm.Span)(old).End()
	}
}

// ClientOption sets options for tracing client requests.
type ClientOption func(*roundTripper)

// captureSearchStatement captures the search URI query or request body.
//
// If the request must be modified (i.e. because the body must be read),
// then captureSearchStatement returns a new *http.Request to be passed
// to the underlying http.RoundTripper. Otherwise, req is returned.
func captureSearchStatement(req *http.Request) (string, *http.Request) {
	if !isSearchURL(req.URL) {
		return "", req
	}

	// If "q" is in query params, use that for statement.
	if req.URL.RawQuery != "" {
		query := req.URL.Query()
		if statement := query.Get("q"); statement != "" {
			return statement, req
		}
	}
	if req.Body == nil || req.Body == http.NoBody {
		return "", req
	}

	var bodyBuf bytes.Buffer
	if req.GetBody != nil {
		// req.GetBody is defined, so we can read a copy of the
		// request body instead of messing with the original request
		// body.
		body, err := req.GetBody()
		if err != nil {
			return "", req
		}
		if _, err := bodyBuf.ReadFrom(limitedBody(body, req.ContentLength)); err != nil {
			body.Close()
			return "", req
		}
		if err := body.Close(); err != nil {
			return "", req
		}
	} else {
		type readCloser struct {
			io.Reader
			io.Closer
		}
		newBody := &readCloser{Closer: req.Body}
		reqCopy := *req
		reqCopy.Body = newBody
		if _, err := bodyBuf.ReadFrom(limitedBody(req.Body, req.ContentLength)); err != nil {
			// Continue with the request, ensuring that req.Body returns
			// the same content and error, but don't use the consumed body
			// for the statement.
			newBody.Reader = io.MultiReader(bytes.NewReader(bodyBuf.Bytes()), errorReader{err: err})
			return "", &reqCopy
		}
		newBody.Reader = io.MultiReader(bytes.NewReader(bodyBuf.Bytes()), req.Body)
		req = &reqCopy
	}

	var statement string
	if req.Header.Get("Content-Encoding") == "gzip" {
		if r, err := gzip.NewReader(&bodyBuf); err == nil {
			if content, err := ioutil.ReadAll(r); err == nil {
				statement = string(content)
			}
		}
	} else {
		statement = bodyBuf.String()
	}
	return statement, req
}

func isSearchURL(url *url.URL) bool {
	switch dir, file := path.Split(url.Path); file {
	case "_search", "_msearch", "_rollup_search":
		return true
	case "template":
		if dir == "" {
			return false
		}
		switch _, file := path.Split(dir[:len(dir)-1]); file {
		case "_search", "_msearch":
			// ".../_search/template" or ".../_msearch/template"
			return true
		}
	}
	return false
}

func limitedBody(r io.Reader, n int64) io.Reader {
	// maxLimit is the maximum size of the request body that we'll read,
	// set to 10000 to match the maximum length of the "db.statement"
	// span context field.
	const maxLimit = 10000
	if n <= 0 {
		return r
	}
	if n > maxLimit {
		n = maxLimit
	}
	return &io.LimitedReader{R: r, N: n}
}

type errorReader struct {
	err error
}

func (r errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}
