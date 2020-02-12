// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"
)

func newScrollFunc(t Transport) Scroll {
	return func(o ...func(*ScrollRequest)) (*Response, error) {
		var r = ScrollRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// Scroll allows to retrieve a large numbers of results from a single search request.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/search-request-scroll.html.
//
type Scroll func(o ...func(*ScrollRequest)) (*Response, error)

// ScrollRequest configures the Scroll API request.
//
type ScrollRequest struct {
	Body io.Reader

	ScrollID           string
	RestTotalHitsAsInt *bool
	Scroll             time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r ScrollRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_search") + 1 + len("scroll") + 1 + len(r.ScrollID))
	path.WriteString("/")
	path.WriteString("_search")
	path.WriteString("/")
	path.WriteString("scroll")
	if r.ScrollID != "" {
		path.WriteString("/")
		path.WriteString(r.ScrollID)
	}

	params = make(map[string]string)

	if r.RestTotalHitsAsInt != nil {
		params["rest_total_hits_as_int"] = strconv.FormatBool(*r.RestTotalHitsAsInt)
	}

	if r.Scroll != 0 {
		params["scroll"] = time.Duration(r.Scroll * time.Millisecond).String()
	}

	if r.ScrollID != "" {
		params["scroll_id"] = r.ScrollID
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
func (f Scroll) WithContext(v context.Context) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.ctx = v
	}
}

// WithScrollID - the scroll ID.
//
func (f Scroll) WithScrollID(v string) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.ScrollID = v
	}
}

// WithBody - The scroll ID if not passed by URL or query parameter..
//
func (f Scroll) WithBody(v io.Reader) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.Body = v
	}
}

// WithRestTotalHitsAsInt - indicates whether hits.total should be rendered as an integer or an object in the rest search response.
//
func (f Scroll) WithRestTotalHitsAsInt(v bool) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.RestTotalHitsAsInt = &v
	}
}

// WithScroll - specify how long a consistent view of the index should be maintained for scrolled search.
//
func (f Scroll) WithScroll(v time.Duration) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.Scroll = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f Scroll) WithPretty() func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f Scroll) WithHuman() func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f Scroll) WithErrorTrace() func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f Scroll) WithFilterPath(v ...string) func(*ScrollRequest) {
	return func(r *ScrollRequest) {
		r.FilterPath = v
	}
}
