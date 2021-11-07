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

// +build !go1.10

package apmhttp

import "net/http"

// UnknownRouteRequestName returns the transaction name for the server request, req,
// when the route could not be determined.
func UnknownRouteRequestName(req *http.Request) string {
	return req.Method + " unknown route"
}

// ServerRequestName returns the transaction name for the server request, req.
func ServerRequestName(req *http.Request) string {
	buf := make([]byte, len(req.Method)+len(req.URL.Path)+1)
	n := copy(buf, req.Method)
	buf[n] = ' '
	copy(buf[n+1:], req.URL.Path)
	return string(buf)
}

// ClientRequestName returns the span name for the client request, req.
func ClientRequestName(req *http.Request) string {
	buf := make([]byte, len(req.Method)+len(req.URL.Host)+1)
	n := copy(buf, req.Method)
	buf[n] = ' '
	copy(buf[n+1:], req.URL.Host)
	return string(buf)
}
