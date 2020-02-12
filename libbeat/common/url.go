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

package common

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var hasScheme = regexp.MustCompile(`^([a-z][a-z0-9+\-.]*)://`)

// MakeURL creates the url based on the url configuration.
// Adds missing parts with defaults (scheme, host, port)
func MakeURL(defaultScheme string, defaultPath string, rawURL string, defaultPort int) (string, error) {
	if defaultScheme == "" {
		defaultScheme = "http"
	}

	if !hasScheme.MatchString(rawURL) {
		rawURL = fmt.Sprintf("%v://%v", defaultScheme, rawURL)
	}

	addr, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	scheme := addr.Scheme
	host := addr.Host
	port := strconv.Itoa(defaultPort)

	if host == "" {
		host = "localhost"
	} else {

		// split host and optional port
		if splitHost, splitPort, err := net.SplitHostPort(host); err == nil {
			host = splitHost
			port = splitPort
		}

		// Check if ipv6
		if strings.Count(host, ":") > 1 && strings.Count(host, "]") == 0 {
			host = "[" + host + "]"
		}
	}

	// Assign default path if not set
	if addr.Path == "" {
		addr.Path = defaultPath
	}

	// reconstruct url
	addr.Scheme = scheme
	addr.Host = host + ":" + port

	return addr.String(), nil
}

func EncodeURLParams(url string, params url.Values) string {
	if len(params) == 0 {
		return url
	}

	return strings.Join([]string{url, "?", params.Encode()}, "")
}
