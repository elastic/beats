// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"strconv"
	"strings"
)

func newIndicesExistsTypeFunc(t Transport) IndicesExistsType {
	return func(index []string, o ...func(*IndicesExistsTypeRequest)) (*Response, error) {
		var r = IndicesExistsTypeRequest{Index: index}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// IndicesExistsType returns information about whether a particular document type exists. (DEPRECATED)
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/indices-types-exists.html.
//
type IndicesExistsType func(index []string, o ...func(*IndicesExistsTypeRequest)) (*Response, error)

// IndicesExistsTypeRequest configures the Indices  Exists Type API request.
//
type IndicesExistsTypeRequest struct {
	Index        []string
	DocumentType []string

	AllowNoIndices    *bool
	ExpandWildcards   string
	IgnoreUnavailable *bool
	Local             *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r IndicesExistsTypeRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "HEAD"

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len("_mapping") + 1 + len(strings.Join(r.DocumentType, ",")))
	path.WriteString("/")
	path.WriteString(strings.Join(r.Index, ","))
	path.WriteString("/")
	path.WriteString("_mapping")
	path.WriteString("/")
	path.WriteString(strings.Join(r.DocumentType, ","))

	params = make(map[string]string)

	if r.AllowNoIndices != nil {
		params["allow_no_indices"] = strconv.FormatBool(*r.AllowNoIndices)
	}

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
	}

	if r.Local != nil {
		params["local"] = strconv.FormatBool(*r.Local)
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

	req, _ := newRequest(method, path.String(), nil)

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
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
func (f IndicesExistsType) WithContext(v context.Context) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.ctx = v
	}
}

// WithDocumentType - a list of document types to check.
//
func (f IndicesExistsType) WithDocumentType(v ...string) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.DocumentType = v
	}
}

// WithAllowNoIndices - whether to ignore if a wildcard indices expression resolves into no concrete indices. (this includes `_all` string or when no indices have been specified).
//
func (f IndicesExistsType) WithAllowNoIndices(v bool) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.AllowNoIndices = &v
	}
}

// WithExpandWildcards - whether to expand wildcard expression to concrete indices that are open, closed or both..
//
func (f IndicesExistsType) WithExpandWildcards(v string) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.ExpandWildcards = v
	}
}

// WithIgnoreUnavailable - whether specified concrete indices should be ignored when unavailable (missing or closed).
//
func (f IndicesExistsType) WithIgnoreUnavailable(v bool) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithLocal - return local information, do not retrieve the state from master node (default: false).
//
func (f IndicesExistsType) WithLocal(v bool) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.Local = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f IndicesExistsType) WithPretty() func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f IndicesExistsType) WithHuman() func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f IndicesExistsType) WithErrorTrace() func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f IndicesExistsType) WithFilterPath(v ...string) func(*IndicesExistsTypeRequest) {
	return func(r *IndicesExistsTypeRequest) {
		r.FilterPath = v
	}
}
