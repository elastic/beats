// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote

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
