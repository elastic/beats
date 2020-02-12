// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"strings"
	"time"
)

func newIngestPutPipelineFunc(t Transport) IngestPutPipeline {
	return func(id string, body io.Reader, o ...func(*IngestPutPipelineRequest)) (*Response, error) {
		var r = IngestPutPipelineRequest{DocumentID: id, Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// IngestPutPipeline creates or updates a pipeline.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/plugins/master/ingest.html.
//
type IngestPutPipeline func(id string, body io.Reader, o ...func(*IngestPutPipelineRequest)) (*Response, error)

// IngestPutPipelineRequest configures the Ingest  Put Pipeline API request.
//
type IngestPutPipelineRequest struct {
	DocumentID string
	Body       io.Reader

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
func (r IngestPutPipelineRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_ingest") + 1 + len("pipeline") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_ingest")
	path.WriteString("/")
	path.WriteString("pipeline")
	path.WriteString("/")
	path.WriteString(r.DocumentID)

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
func (f IngestPutPipeline) WithContext(v context.Context) func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
//
func (f IngestPutPipeline) WithMasterTimeout(v time.Duration) func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - explicit operation timeout.
//
func (f IngestPutPipeline) WithTimeout(v time.Duration) func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f IngestPutPipeline) WithPretty() func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f IngestPutPipeline) WithHuman() func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f IngestPutPipeline) WithErrorTrace() func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f IngestPutPipeline) WithFilterPath(v ...string) func(*IngestPutPipelineRequest) {
	return func(r *IngestPutPipelineRequest) {
		r.FilterPath = v
	}
}
