// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strings"
	"time"
)

func newPutScriptFunc(t Transport) PutScript {
	return func(id string, body io.Reader, o ...func(*PutScriptRequest)) (*Response, error) {
		var r = PutScriptRequest{DocumentID: id, Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// PutScript creates or updates a script.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/modules-scripting.html.
//
type PutScript func(id string, body io.Reader, o ...func(*PutScriptRequest)) (*Response, error)

// PutScriptRequest configures the Put Script API request.
//
type PutScriptRequest struct {
	DocumentID string
	Body       io.Reader

	ScriptContext string
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
func (r PutScriptRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_scripts") + 1 + len(r.DocumentID) + 1 + len(r.ScriptContext))
	path.WriteString("/")
	path.WriteString("_scripts")
	path.WriteString("/")
	path.WriteString(r.DocumentID)
	if r.ScriptContext != "" {
		path.WriteString("/")
		path.WriteString(r.ScriptContext)
	}

	params = make(map[string]string)

	if r.ScriptContext != "" {
		params["context"] = r.ScriptContext
	}

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
func (f PutScript) WithContext(v context.Context) func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.ctx = v
	}
}

// WithScriptContext - script context.
//
func (f PutScript) WithScriptContext(v string) func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.ScriptContext = v
	}
}

// WithMasterTimeout - specify timeout for connection to master.
//
func (f PutScript) WithMasterTimeout(v time.Duration) func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - explicit operation timeout.
//
func (f PutScript) WithTimeout(v time.Duration) func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f PutScript) WithPretty() func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f PutScript) WithHuman() func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f PutScript) WithErrorTrace() func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f PutScript) WithFilterPath(v ...string) func(*PutScriptRequest) {
	return func(r *PutScriptRequest) {
		r.FilterPath = v
	}
}
