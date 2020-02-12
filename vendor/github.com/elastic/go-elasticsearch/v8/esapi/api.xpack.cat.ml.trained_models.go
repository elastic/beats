// Licensed to Elasticsearch B.V under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
//
// Code generated from specification version 8.0.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

func newCatMLTrainedModelsFunc(t Transport) CatMLTrainedModels {
	return func(o ...func(*CatMLTrainedModelsRequest)) (*Response, error) {
		var r = CatMLTrainedModelsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// CatMLTrainedModels -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-inference-stats.html.
//
type CatMLTrainedModels func(o ...func(*CatMLTrainedModelsRequest)) (*Response, error)

// CatMLTrainedModelsRequest configures the CatML Trained Models API request.
//
type CatMLTrainedModelsRequest struct {
	ModelID string

	AllowNoMatch *bool
	Bytes        string
	Format       string
	From         *int
	H            []string
	Help         *bool
	S            []string
	Size         *int
	Time         string
	V            *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r CatMLTrainedModelsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_cat") + 1 + len("ml") + 1 + len("trained_models") + 1 + len(r.ModelID))
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("ml")
	path.WriteString("/")
	path.WriteString("trained_models")
	if r.ModelID != "" {
		path.WriteString("/")
		path.WriteString(r.ModelID)
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Bytes != "" {
		params["bytes"] = r.Bytes
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if len(r.H) > 0 {
		params["h"] = strings.Join(r.H, ",")
	}

	if r.Help != nil {
		params["help"] = strconv.FormatBool(*r.Help)
	}

	if len(r.S) > 0 {
		params["s"] = strings.Join(r.S, ",")
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if r.Time != "" {
		params["time"] = r.Time
	}

	if r.V != nil {
		params["v"] = strconv.FormatBool(*r.V)
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

	req, err := newRequest(method, path.String(), nil)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
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
func (f CatMLTrainedModels) WithContext(v context.Context) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.ctx = v
	}
}

// WithModelID - the ID of the trained models stats to fetch.
//
func (f CatMLTrainedModels) WithModelID(v string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.ModelID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no trained models. (this includes `_all` string or when no trained models have been specified).
//
func (f CatMLTrainedModels) WithAllowNoMatch(v bool) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithBytes - the unit in which to display byte values.
//
func (f CatMLTrainedModels) WithBytes(v string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Bytes = v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
//
func (f CatMLTrainedModels) WithFormat(v string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Format = v
	}
}

// WithFrom - skips a number of trained models.
//
func (f CatMLTrainedModels) WithFrom(v int) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.From = &v
	}
}

// WithH - comma-separated list of column names to display.
//
func (f CatMLTrainedModels) WithH(v ...string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
//
func (f CatMLTrainedModels) WithHelp(v bool) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Help = &v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
//
func (f CatMLTrainedModels) WithS(v ...string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.S = v
	}
}

// WithSize - specifies a max number of trained models to get.
//
func (f CatMLTrainedModels) WithSize(v int) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Size = &v
	}
}

// WithTime - the unit in which to display time values.
//
func (f CatMLTrainedModels) WithTime(v string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
//
func (f CatMLTrainedModels) WithV(v bool) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f CatMLTrainedModels) WithPretty() func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f CatMLTrainedModels) WithHuman() func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f CatMLTrainedModels) WithErrorTrace() func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f CatMLTrainedModels) WithFilterPath(v ...string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f CatMLTrainedModels) WithHeader(h map[string]string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
//
func (f CatMLTrainedModels) WithOpaqueID(s string) func(*CatMLTrainedModelsRequest) {
	return func(r *CatMLTrainedModelsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
