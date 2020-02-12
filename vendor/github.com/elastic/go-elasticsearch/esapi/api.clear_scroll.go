// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strings"
)

func newClearScrollFunc(t Transport) ClearScroll {
	return func(o ...func(*ClearScrollRequest)) (*Response, error) {
		var r = ClearScrollRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// ClearScroll explicitely clears the search context for a scroll.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/search-request-scroll.html.
//
type ClearScroll func(o ...func(*ClearScrollRequest)) (*Response, error)

// ClearScrollRequest configures the Clear Scroll API request.
//
type ClearScrollRequest struct {
	Body io.Reader

	ScrollID []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r ClearScrollRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_search") + 1 + len("scroll") + 1 + len(strings.Join(r.ScrollID, ",")))
	path.WriteString("/")
	path.WriteString("_search")
	path.WriteString("/")
	path.WriteString("scroll")
	if len(r.ScrollID) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.ScrollID, ","))
	}

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
func (f ClearScroll) WithContext(v context.Context) func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.ctx = v
	}
}

// WithScrollID - a list of scroll ids to clear.
//
func (f ClearScroll) WithScrollID(v ...string) func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.ScrollID = v
	}
}

// WithBody - A comma-separated list of scroll IDs to clear if none was specified via the scroll_id parameter.
//
func (f ClearScroll) WithBody(v io.Reader) func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.Body = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f ClearScroll) WithPretty() func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f ClearScroll) WithHuman() func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f ClearScroll) WithErrorTrace() func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f ClearScroll) WithFilterPath(v ...string) func(*ClearScrollRequest) {
	return func(r *ClearScrollRequest) {
		r.FilterPath = v
	}
}
