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
	"fmt"
	"net"
	"net/netip"
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

	// Since Go 1.26, url.Parse rejects unbracketed IPv6 addresses in the host,
	// so bracket them before parsing to keep accepting inputs like "2001:db8::1".
	rawURL = bracketIPv6Host(rawURL)

	addr, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	scheme := addr.Scheme
	host := addr.Host
	port := ""
	if defaultPort > 0 {
		port = strconv.Itoa(defaultPort)
	}

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
	addr.Host = host
	if port != "" {
		addr.Host += ":" + port
	}

	return addr.String(), nil
}

// bracketIPv6Host wraps an unbracketed IPv6 address in the host subcomponent
// of rawURL in square brackets, e.g. "http://2001:db8::1/path" becomes
// "http://[2001:db8::1]/path". Any other input is returned unchanged.
func bracketIPv6Host(rawURL string) string {
	schemeEnd := strings.Index(rawURL, "://")
	if schemeEnd < 0 {
		return rawURL
	}
	authStart := schemeEnd + len("://")

	authEnd := strings.IndexAny(rawURL[authStart:], "/?#")
	if authEnd < 0 {
		authEnd = len(rawURL)
	} else {
		authEnd += authStart
	}

	// Split optional userinfo from the host.
	userinfo, host := "", rawURL[authStart:authEnd]
	if at := strings.LastIndex(host, "@"); at >= 0 {
		userinfo, host = host[:at+1], host[at+1:]
	}

	// Already bracketed, or not enough colons to be an IPv6 address.
	if strings.Contains(host, "[") || strings.Count(host, ":") < 2 {
		return rawURL
	}
	if _, err := netip.ParseAddr(host); err != nil {
		return rawURL
	}

	return rawURL[:authStart] + userinfo + "[" + host + "]" + rawURL[authEnd:]
}

func EncodeURLParams(url string, params url.Values) string {
	if len(params) == 0 {
		return url
	}

	return strings.Join([]string{url, "?", params.Encode()}, "")
}

type ParseHint func(raw string) string

// ParseURL tries to parse a URL and return the parsed result.
func ParseURL(raw string, hints ...ParseHint) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}

	if len(hints) == 0 {
		hints = append(hints, WithDefaultScheme("http"))
	}

	if !strings.Contains(raw, "://") {
		for _, hint := range hints {
			raw = hint(raw)
		}
	}

	return url.Parse(raw)
}

func WithDefaultScheme(scheme string) ParseHint {
	return func(raw string) string {
		if !strings.Contains(raw, "://") {
			return scheme + "://" + raw
		}
		return raw
	}
}
