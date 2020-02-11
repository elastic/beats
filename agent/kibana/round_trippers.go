// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kibana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// UserAgentRoundTripper adds a User-Agent string on every request.
type UserAgentRoundTripper struct {
	rt        http.RoundTripper
	userAgent string
}

// RoundTrip adds a User-Agent string on every request if its not already present.
func (r *UserAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	const userAgentHeader = "User-Agent"
	if len(req.Header.Get(userAgentHeader)) == 0 {
		req.Header.Set(userAgentHeader, r.userAgent)
	}

	return r.rt.RoundTrip(req)
}

// NewUserAgentRoundTripper returns a new UserAgentRoundTripper.
func NewUserAgentRoundTripper(wrapped http.RoundTripper, userAgent string) http.RoundTripper {
	return &UserAgentRoundTripper{rt: wrapped, userAgent: userAgent}
}

// DebugRoundTripper is a debugging RoundTripper that can be inserted in the chain of existing
// http.RoundTripper. This will output to the specific logger at debug level the request and response
// information for each calls. This is most useful in development or when debugging any calls
// between the agent and the Fleet API.
type DebugRoundTripper struct {
	rt  http.RoundTripper
	log debugLogger
}

type debugLogger interface {
	Debug(args ...interface{})
}

// RoundTrip send the raw request and raw response from the client into the logger at debug level.
// This should not be used in a production environment because it will leak credentials.
func (r *DebugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Note: I could use httputil.DumpResponse here, but I want to make sure I can pretty print
	// the response of the body when I receive a JSON response.
	var b strings.Builder

	b.WriteString("Request:\n")
	b.WriteString("  Verb: " + req.Method + "\n")
	b.WriteString("  URI: " + req.URL.RequestURI() + "\n")
	b.WriteString("  Headers:\n")

	for k, v := range req.Header {
		b.WriteString("     key: " + k + " values: {" + strings.Join(v, ", ") + "}\n")
	}

	if req.Body != nil {
		dataReq, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, errors.Wrap(err, "fail to read the body of the request")
		}
		req.Body.Close()

		req.Body = ioutil.NopCloser(bytes.NewBuffer(dataReq))

		b.WriteString("Request Body:\n")
		b.WriteString(string(prettyBody(dataReq)) + "\n")
	}

	startTime := time.Now()
	resp, err := r.rt.RoundTrip(req)

	duration := time.Since(startTime)

	b.WriteString("Response:\n")
	b.WriteString("  Headers:\n")

	for k, v := range resp.Header {
		b.WriteString("     key: " + k + " values: {" + strings.Join(v, ", ") + "}\n")
	}

	b.WriteString(fmt.Sprintf("  Response code: %d\n", resp.StatusCode))
	b.WriteString(fmt.Sprintf("Request executed in %dms\n", duration.Nanoseconds()/int64(time.Millisecond)))

	// If the body is empty we just return
	if resp.Body == nil {
		r.log.Debug(b.String())
		return resp, err
	}

	// Hijack the body and output it in the log, this is only for debugging and development.
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, errors.Wrap(err, "fail to read the body of the response")
	}
	resp.Body.Close()

	b.WriteString("Response Body:\n")
	b.WriteString(string(prettyBody(data)) + "\n")

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	r.log.Debug(b.String())

	return resp, err
}

// NewDebugRoundTripper wraps an existing http.RoundTripper into a DebugRoundTripper that will log
// the call executed to the service.
func NewDebugRoundTripper(wrapped http.RoundTripper, log debugLogger) http.RoundTripper {
	return &DebugRoundTripper{rt: wrapped, log: log}
}

// EnforceKibanaVersionRoundTripper sets the kbn-version header on every request.
type EnforceKibanaVersionRoundTripper struct {
	rt      http.RoundTripper
	version string
}

// RoundTrip adds the kbn-version header, if the remote kibana is not equal or superior the call
/// will fail.
func (r *EnforceKibanaVersionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	const key = "kbn-version"
	req.Header.Set(key, r.version)
	return r.rt.RoundTrip(req)
}

// NewEnforceKibanaVersionRoundTripper enforce the remove endpoint to be a a certain version, if the
// remove kibana is not equal or superior on the requested version the call will fail.
func NewEnforceKibanaVersionRoundTripper(wrapped http.RoundTripper, version string) http.RoundTripper {
	return &EnforceKibanaVersionRoundTripper{rt: wrapped, version: version}
}

// BasicAuthRoundTripper wraps any request using a basic auth.
type BasicAuthRoundTripper struct {
	rt       http.RoundTripper
	username string
	password string
}

// RoundTrip add username and password on every request send to the remove service.
func (r *BasicAuthRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	// if we already have authorization set on the request we do not force our username, password.
	const key = "Authorization"

	if len(req.Header.Get(key)) > 0 {
		return r.rt.RoundTrip(req)
	}

	req.SetBasicAuth(r.username, r.password)
	return r.rt.RoundTrip(req)
}

// NewBasicAuthRoundTripper returns a Basic Auth round tripper.
func NewBasicAuthRoundTripper(
	wrapped http.RoundTripper,
	username, password string,
) http.RoundTripper {
	return &BasicAuthRoundTripper{rt: wrapped, username: username, password: password}
}

func prettyBody(data []byte) []byte {
	var pretty bytes.Buffer

	if err := json.Indent(&pretty, data, "", " "); err != nil {
		// indent doesn't valid the JSON when it parses it, we assume that if the
		// buffer is empty we failed to indent anything and we just return the raw string.
		if pretty.Len() > 0 {
			return pretty.Bytes()
		}
	}

	return data
}
