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
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/conditions"
)

type Config struct {
	URLs         []string       `config:"urls"`
	Hosts        []string       `config:"hosts"`
	ProxyURL     string         `config:"proxy_url"`
	Timeout      time.Duration  `config:"timeout"`
	MaxRedirects int            `config:"max_redirects"`
	Response     responseConfig `config:"response"`

	Mode monitors.IPSettings `config:",inline"`

	// authentication
	Username string `config:"username"`
	Password string `config:"password"`

	// configure tls (if not configured HTTPS will use system defaults)
	TLS *tlscommon.Config `config:"ssl"`

	// http(s) ping validation
	Check checkConfig `config:"check"`
}

type responseConfig struct {
	IncludeBody         string `config:"include_body"`
	IncludeBodyMaxBytes int    `config:"include_body_max_bytes"`
}

type checkConfig struct {
	Request  requestParameters  `config:"request"`
	Response responseParameters `config:"response"`
}

type requestParameters struct {
	// HTTP request configuration
	Method      string            `config:"method"`      // http request method
	SendHeaders map[string]string `config:"headers"`     // http request headers
	SendBody    string            `config:"body"`        // send body payload
	Compression compressionConfig `config:"compression"` // optionally compress payload

	// TODO:
	//  - add support for cookies
	//  - select HTTP version. golang lib will either use 1.1 or 2.0 if HTTPS is used, otherwise HTTP 1.1 . => implement/use specific http.RoundTripper implementation to change wire protocol/version being used
}

type responseParameters struct {
	// expected HTTP response configuration
	Status      uint16               `config:"status" verify:"min=0, max=699"`
	RecvHeaders map[string]string    `config:"headers"`
	RecvBody    []match.Matcher      `config:"body"`
	RecvJSON    []*jsonResponseCheck `config:"json"`
}

type jsonResponseCheck struct {
	Description string             `config:"description"`
	Condition   *conditions.Config `config:"condition"`
}

type compressionConfig struct {
	Type  string `config:"type"`
	Level int    `config:"level"`
}

var defaultConfig = Config{
	Timeout:      16 * time.Second,
	MaxRedirects: 0,
	Response: responseConfig{
		IncludeBody:         "on_error",
		IncludeBodyMaxBytes: 2048,
	},
	Mode: monitors.DefaultIPSettings,
	Check: checkConfig{
		Request: requestParameters{
			Method:      "GET",
			SendHeaders: nil,
			SendBody:    "",
		},
		Response: responseParameters{
			Status:      0,
			RecvHeaders: nil,
			RecvBody:    []match.Matcher{},
			RecvJSON:    nil,
		},
	},
}

// Validate validates of the responseConfig object is valid or not
func (r *responseConfig) Validate() error {
	switch strings.ToLower(r.IncludeBody) {
	case "always", "on_error", "never":
	default:
		return fmt.Errorf("unknown option for `include_body`: '%s', please use one of 'always', 'on_error', 'never'", r.IncludeBody)
	}

	if r.IncludeBodyMaxBytes <= 0 {
		return fmt.Errorf("include_body_max_bytes must be a positive integer, got %d", r.IncludeBodyMaxBytes)
	}

	return nil
}

// Validate validates of the requestParameters object is valid or not
func (r *requestParameters) Validate() error {
	switch strings.ToUpper(r.Method) {
	case "HEAD", "GET", "POST":
	default:
		return fmt.Errorf("HTTP method '%v' not supported", r.Method)
	}

	return nil
}

// Validate validates of the compressionConfig object is valid or not
func (c *compressionConfig) Validate() error {
	t := strings.ToLower(c.Type)
	if t != "" && t != "gzip" {
		return fmt.Errorf("compression type '%v' not supported", c.Type)
	}

	if t == "" {
		return nil
	}

	if !(0 <= c.Level && c.Level <= 9) {
		return fmt.Errorf("compression level %v invalid", c.Level)
	}

	return nil
}

// Validate validates of the Config object is valid or not
func (c *Config) Validate() error {
	if len(c.Hosts) == 0 && len(c.URLs) == 0 {
		return fmt.Errorf("hosts is a mandatory parameter")
	}

	if len(c.URLs) != 0 {
		c.Hosts = append(c.Hosts, c.URLs...)
	}

	// updateScheme looks at TLS config to decide if http or https should be used to update the host
	updateScheme := func(host string) string {
		if c.TLS != nil && *c.TLS.Enabled == true {
			return fmt.Sprint("https://", host)
		}
		return fmt.Sprint("http://", host)
	}

	// Check if the URL is not parseable. If yes, then append scheme.
	// If the url is valid but host or scheme is empty which can occur when someone configures host:port
	// then update the scheme there as well.
	for i := 0; i < len(c.Hosts); i++ {
		host := c.Hosts[i]
		u, err := url.ParseRequestURI(host)
		if err != nil {
			c.Hosts[i] = updateScheme(host)
		} else if u.Scheme == "" || u.Host == "" {
			c.Hosts[i] = updateScheme(host)
		}
	}

	return nil
}
