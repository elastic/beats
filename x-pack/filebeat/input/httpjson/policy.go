// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (

	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

// Evaluate is a template expression evaluation function which accepts a
// valid go text/template expression and evaluates the expected field value to the
// field value present in data using the defined operator/function in the given expression.
// Example : [[ eq .last_response.body.status "completed" ]] -- which means here data is a http response
// containing a field "status" under the field "body" , and value status should be equal to the string "completed"
type Evaluate func(expression *valueTpl, data []byte, log *logp.Logger) (bool, error)

// Policy is responsible for maintaining different http client policies
// Currently just contains a retry policy function
type Policy struct {
	fn         Evaluate
	expression *valueTpl
	log        *logp.Logger
}

func newHTTPPolicy(fn Evaluate, expression *valueTpl, log *logp.Logger) *Policy {
	return &Policy{
		fn:         fn,
		expression: expression,
		log:        log,
	}
}

// CustomRetryPolicy provides a custom callback for Client.CheckRetry, which
// will retry on connection errors and server errors.
func (p *Policy) CustomRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		var v *url.Error
		if errors.As(err, &v) {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			var k x509.UnknownAuthorityError
			if errors.As(v.Err, &k) {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	// Evaluate custom expression
	if p.fn != nil && p.expression != nil {
		var retry bool

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return retry, fmt.Errorf("failed to read http response body : %w", err)
		}

		err = resp.Body.Close()
		if err != nil {
			return retry, fmt.Errorf("error closing response body : %w", err)
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		result, err := p.fn(p.expression, body, p.log)
		if err != nil {
			return retry, err
		}

		if !result {
			retry = true
		}

		return retry, nil
	}

	return false, nil
}
