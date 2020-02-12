// Code generated from specification version 7.0.0 (5e798c1): DO NOT EDIT

package esapi

import (
	"context"
	"strings"
	"time"
)

func newSnapshotVerifyRepositoryFunc(t Transport) SnapshotVerifyRepository {
	return func(repository string, o ...func(*SnapshotVerifyRepositoryRequest)) (*Response, error) {
		var r = SnapshotVerifyRepositoryRequest{Repository: repository}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SnapshotVerifyRepository verifies a repository.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/master/modules-snapshots.html.
//
type SnapshotVerifyRepository func(repository string, o ...func(*SnapshotVerifyRepositoryRequest)) (*Response, error)

// SnapshotVerifyRepositoryRequest configures the Snapshot  Verify Repository API request.
//
type SnapshotVerifyRepositoryRequest struct {
	Repository    string
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
func (r SnapshotVerifyRepositoryRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_snapshot") + 1 + len(r.Repository) + 1 + len("_verify"))
	path.WriteString("/")
	path.WriteString("_snapshot")
	path.WriteString("/")
	path.WriteString(r.Repository)
	path.WriteString("/")
	path.WriteString("_verify")

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
func (f SnapshotVerifyRepository) WithContext(v context.Context) func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
//
func (f SnapshotVerifyRepository) WithMasterTimeout(v time.Duration) func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - explicit operation timeout.
//
func (f SnapshotVerifyRepository) WithTimeout(v time.Duration) func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SnapshotVerifyRepository) WithPretty() func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SnapshotVerifyRepository) WithHuman() func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SnapshotVerifyRepository) WithErrorTrace() func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SnapshotVerifyRepository) WithFilterPath(v ...string) func(*SnapshotVerifyRepositoryRequest) {
	return func(r *SnapshotVerifyRepositoryRequest) {
		r.FilterPath = v
	}
}
