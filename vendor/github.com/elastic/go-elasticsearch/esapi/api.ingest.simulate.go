// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strconv"
	"strings"
)

func newIngestSimulateFunc(t Transport) IngestSimulate {
	return func(body io.Reader, o ...func(*IngestSimulateRequest)) (*Response, error) {
		var r = IngestSimulateRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// IngestSimulate allows to simulate a pipeline with example documents.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/plugins/master/ingest.html.
//
type IngestSimulate func(body io.Reader, o ...func(*IngestSimulateRequest)) (*Response, error)

// IngestSimulateRequest configures the Ingest Simulate API request.
//
type IngestSimulateRequest struct {
	DocumentID string
	Body       io.Reader

	Verbose *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r IngestSimulateRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_ingest") + 1 + len("pipeline") + 1 + len(r.DocumentID) + 1 + len("_simulate"))
	path.WriteString("/")
	path.WriteString("_ingest")
	path.WriteString("/")
	path.WriteString("pipeline")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
	}
	path.WriteString("/")
	path.WriteString("_simulate")

	params = make(map[string]string)

	if r.Verbose != nil {
		params["verbose"] = strconv.FormatBool(*r.Verbose)
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
func (f IngestSimulate) WithContext(v context.Context) func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.ctx = v
	}
}

// WithDocumentID - pipeline ID.
//
func (f IngestSimulate) WithDocumentID(v string) func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.DocumentID = v
	}
}

// WithVerbose - verbose mode. display data output for each processor in executed pipeline.
//
func (f IngestSimulate) WithVerbose(v bool) func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.Verbose = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f IngestSimulate) WithPretty() func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f IngestSimulate) WithHuman() func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f IngestSimulate) WithErrorTrace() func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f IngestSimulate) WithFilterPath(v ...string) func(*IngestSimulateRequest) {
	return func(r *IngestSimulateRequest) {
		r.FilterPath = v
	}
}
