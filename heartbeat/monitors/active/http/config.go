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
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/conditions"

	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"

	"github.com/elastic/beats/heartbeat/monitors"
)

type Config struct {
	URLs         []string      `config:"urls" validate:"required"`
	ProxyURL     string        `config:"proxy_url"`
	Timeout      time.Duration `config:"timeout"`
	MaxRedirects int           `config:"max_redirects"`

	Mode monitors.IPSettings `config:",inline"`

	// authentication
	Username string `config:"username"`
	Password string `config:"password"`

	// configure tls (if not configured HTTPS will use system defaults)
	TLS *tlscommon.Config `config:"ssl"`

	// http(s) ping validation
	Check checkConfig `config:"check"`
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
	MaxRedirects: 10,
	Mode:         monitors.DefaultIPSettings,
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

func (r *requestParameters) Validate() error {
	switch strings.ToUpper(r.Method) {
	case "HEAD", "GET", "POST":
	default:
		return fmt.Errorf("HTTP method '%v' not supported", r.Method)
	}

	return nil
}

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
