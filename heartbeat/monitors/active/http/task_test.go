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

package http

import (
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitHostnamePort(t *testing.T) {
	var urlTests = []struct {
		name          string
		scheme        string
		host          string
		expectedHost  string
		expectedPort  uint16
		expectedError error
	}{
		{
			"plain",
			"http",
			"foo",
			"foo",
			80,
			nil,
		},
		{
			"dotted domain",
			"http",
			"www.foo.com",
			"www.foo.com",
			80,
			nil,
		},
		{
			"dotted domain, custom port",
			"http",
			"www.foo.com:8080",
			"www.foo.com",
			8080,
			nil,
		},
		{
			"https plain",
			"https",
			"foo",
			"foo",
			443,
			nil,
		},
		{
			"custom port",
			"http",
			"foo:81",
			"foo",
			81,
			nil,
		},
		{
			"https custom port",
			"https",
			"foo:444",
			"foo",
			444,
			nil,
		},
		{
			"bad scheme",
			"httpz",
			"foo",
			"foo",
			81,
			&net.AddrError{},
		},
	}
	for _, test := range urlTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			url := &url.URL{
				Scheme: test.scheme,
				Host:   test.host,
			}
			request := &http.Request{
				URL: url,
			}
			host, port, err := splitHostnamePort(request)

			if err != nil {
				if test.expectedError == nil {
					t.Error(err)
				} else if reflect.TypeOf(err) != reflect.TypeOf(test.expectedError) {
					t.Errorf("Expected %T but got %T", err, test.expectedError)
				}
			} else {
				if host != test.expectedHost {
					t.Errorf("Unexpected host for %#v: expected %q, got %q", request, test.expectedHost, host)
				}
				if port != test.expectedPort {
					t.Errorf("Unexpected port for %#v: expected %q, got %q", request, test.expectedPort, port)
				}
			}

		})
	}
}

func makeTestHTTPRequest(t *testing.T) *http.Request {
	req, err := http.NewRequest("GET", "http://example.net", nil)
	assert.Nil(t, err)
	return req
}

func TestZeroMaxRedirectShouldError(t *testing.T) {
	checker := makeCheckRedirect(0)
	req := makeTestHTTPRequest(t)

	res := checker(req, nil)
	assert.Equal(t, http.ErrUseLastResponse, res)
}

func TestNonZeroRedirect(t *testing.T) {
	limit := 5
	checker := makeCheckRedirect(limit)

	var via []*http.Request
	// Test requests within the limit
	for i := 0; i < limit; i++ {
		req := makeTestHTTPRequest(t)
		assert.Nil(t, checker(req, via))
		via = append(via, req)
	}

	// We are now at the limit, this request should fail
	assert.Equal(t, http.ErrUseLastResponse, checker(makeTestHTTPRequest(t), via))
}
