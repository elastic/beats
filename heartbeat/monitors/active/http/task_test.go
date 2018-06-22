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
)

func TestSplitHostnamePort(t *testing.T) {
	var urlTests = []struct {
		scheme        string
		host          string
		expectedHost  string
		expectedPort  uint16
		expectedError error
	}{
		{
			"http",
			"foo",
			"foo",
			80,
			nil,
		},
		{
			"http",
			"www.foo.com",
			"www.foo.com",
			80,
			nil,
		},
		{
			"http",
			"www.foo.com:8080",
			"www.foo.com",
			8080,
			nil,
		},
		{
			"https",
			"foo",
			"foo",
			443,
			nil,
		},
		{
			"http",
			"foo:81",
			"foo",
			81,
			nil,
		},
		{
			"https",
			"foo:444",
			"foo",
			444,
			nil,
		},
		{
			"httpz",
			"foo",
			"foo",
			81,
			&net.AddrError{},
		},
	}
	for _, test := range urlTests {
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
			continue
		}
		if host != test.expectedHost {
			t.Errorf("Unexpected host for %#v: expected %q, got %q", request, test.expectedHost, host)
		}
		if port != test.expectedPort {
			t.Errorf("Unexpected port for %#v: expected %q, got %q", request, test.expectedPort, port)
		}
	}
}
