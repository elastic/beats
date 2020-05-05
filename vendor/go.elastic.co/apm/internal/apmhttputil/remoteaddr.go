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

package apmhttputil

import (
	"net/http"
	"strconv"
)

// RemoteAddr returns the remote (peer) socket address for req,
// a server HTTP request.
func RemoteAddr(req *http.Request) string {
	remoteAddr, _ := splitHost(req.RemoteAddr)
	return remoteAddr
}

// DestinationAddr returns the destination server address and port
// for req, a client HTTP request.
//
// If req.URL.Host contains a port it will be returned, and otherwise
// the default port according to req.URL.Scheme will be returned. If
// the included port is not a valid integer, or no port is included
// and the scheme is unknown, the returned port value will be zero.
func DestinationAddr(req *http.Request) (string, int) {
	host, strport := splitHost(req.URL.Host)
	var port int
	if strport != "" {
		port, _ = strconv.Atoi(strport)
	} else {
		port = SchemeDefaultPort(req.URL.Scheme)
	}
	return host, port
}

// SchemeDefaultPort returns the default port for the given URI scheme,
// if known, or 0 otherwise.
func SchemeDefaultPort(scheme string) int {
	switch scheme {
	case "http":
		return 80
	case "https":
		return 443
	}
	return 0
}
