// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strings"
)

func newRenderSearchTemplateFunc(t Transport) RenderSearchTemplate {
	return func(o ...func(*RenderSearchTemplateRequest)) (*Response, error) {
		var r = RenderSearchTemplateRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// RenderSearchTemplate allows to use the Mustache language to pre-render a search definition.
//
// See full documentation at http://www.elasticsearch.org/guide/en/elasticsearch/reference/master/search-template.html.
//
type RenderSearchTemplate func(o ...func(*RenderSearchTemplateRequest)) (*Response, error)

// RenderSearchTemplateRequest configures the Render Search Template API request.
//
type RenderSearchTemplateRequest struct {
	DocumentID string
	Body       io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r RenderSearchTemplateRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_render") + 1 + len("template") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_render")
	path.WriteString("/")
	path.WriteString("template")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
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
func (f RenderSearchTemplate) WithContext(v context.Context) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.ctx = v
	}
}

// WithDocumentID - the ID of the stored search template.
//
func (f RenderSearchTemplate) WithDocumentID(v string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.DocumentID = v
	}
}

// WithBody - The search definition template and its params.
//
func (f RenderSearchTemplate) WithBody(v io.Reader) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Body = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f RenderSearchTemplate) WithPretty() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f RenderSearchTemplate) WithHuman() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f RenderSearchTemplate) WithErrorTrace() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f RenderSearchTemplate) WithFilterPath(v ...string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.FilterPath = v
	}
}
