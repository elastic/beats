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

func newSecurityGetAPIKeyFunc(t Transport) SecurityGetAPIKey {
	return func(o ...func(*SecurityGetAPIKeyRequest)) (*Response, error) {
		var r = SecurityGetAPIKeyRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SecurityGetAPIKey -
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-get-api-key.html.
//
type SecurityGetAPIKey func(o ...func(*SecurityGetAPIKeyRequest)) (*Response, error)

// SecurityGetAPIKeyRequest configures the Security GetAPI Key API request.
//
type SecurityGetAPIKeyRequest struct {
	ID        string
	Name      string
	Owner     *bool
	RealmName string
	Username  string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SecurityGetAPIKeyRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(len("/_security/api_key"))
	path.WriteString("/_security/api_key")

	params = make(map[string]string)

	if r.ID != "" {
		params["id"] = r.ID
	}

	if r.Name != "" {
		params["name"] = r.Name
	}

	if r.Owner != nil {
		params["owner"] = strconv.FormatBool(*r.Owner)
	}

	if r.RealmName != "" {
		params["realm_name"] = r.RealmName
	}

	if r.Username != "" {
		params["username"] = r.Username
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
func (f SecurityGetAPIKey) WithContext(v context.Context) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.ctx = v
	}
}

// WithID - api key ID of the api key to be retrieved.
//
func (f SecurityGetAPIKey) WithID(v string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.ID = v
	}
}

// WithName - api key name of the api key to be retrieved.
//
func (f SecurityGetAPIKey) WithName(v string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.Name = v
	}
}

// WithOwner - flag to query api keys owned by the currently authenticated user.
//
func (f SecurityGetAPIKey) WithOwner(v bool) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.Owner = &v
	}
}

// WithRealmName - realm name of the user who created this api key to be retrieved.
//
func (f SecurityGetAPIKey) WithRealmName(v string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.RealmName = v
	}
}

// WithUsername - user name of the user who created this api key to be retrieved.
//
func (f SecurityGetAPIKey) WithUsername(v string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.Username = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SecurityGetAPIKey) WithPretty() func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SecurityGetAPIKey) WithHuman() func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SecurityGetAPIKey) WithErrorTrace() func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SecurityGetAPIKey) WithFilterPath(v ...string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SecurityGetAPIKey) WithHeader(h map[string]string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
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
func (f SecurityGetAPIKey) WithOpaqueID(s string) func(*SecurityGetAPIKeyRequest) {
	return func(r *SecurityGetAPIKeyRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
