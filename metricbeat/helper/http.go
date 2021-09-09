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

package helper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/metricbeat/helper/dialer"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var userAgent = useragent.UserAgent("Metricbeat")

// HTTP is a custom HTTP Client that handle the complexity of connection and retrieving information
// from HTTP endpoint.
type HTTP struct {
	hostData mb.HostData
	client   *http.Client // HTTP client that is reused across requests.
	headers  http.Header
	name     string
	uri      string
	method   string
	body     []byte
}

// NewHTTP creates new http helper
func NewHTTP(base mb.BaseMetricSet) (*HTTP, error) {
	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return NewHTTPFromConfig(config, base.HostData())
}

// newHTTPWithConfig creates a new http helper from some configuration
func NewHTTPFromConfig(config Config, hostData mb.HostData) (*HTTP, error) {
	headers := http.Header{}
	if config.Headers == nil {
		config.Headers = map[string]string{}
	}
	for k, v := range config.Headers {
		headers.Set(k, v)
	}

	if config.BearerTokenFile != "" {
		header, err := getAuthHeaderFromToken(config.BearerTokenFile)
		if err != nil {
			return nil, err
		}
		headers.Set("Authorization", header)
	}

	// Ensure backward compatibility
	builder := hostData.Transport
	if builder == nil {
		builder = dialer.NewDefaultDialerBuilder()
	}

	dialer, err := builder.Make(config.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	client, err := config.Transport.Client(
		httpcommon.WithBaseDialer(dialer),
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithHeaderRoundTripper(map[string]string{"User-Agent": userAgent}),
	)
	if err != nil {
		return nil, err
	}

	return &HTTP{
		hostData: hostData,
		client:   client,
		headers:  headers,
		method:   "GET",
		uri:      hostData.SanitizedURI,
		body:     nil,
	}, nil
}

// FetchResponse fetches a response for the http metricset.
// It's important that resp.Body has to be closed if this method is used. Before using this method
// check if one of the other Fetch* methods could be used as they ensure that the Body is properly closed.
func (h *HTTP) FetchResponse() (*http.Response, error) {
	// Create a fresh reader every time
	var reader io.Reader
	if h.body != nil {
		reader = bytes.NewReader(h.body)
	}

	req, err := http.NewRequest(h.method, h.uri, reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}
	req.Header = h.headers
	if h.hostData.User != "" || h.hostData.Password != "" {
		req.SetBasicAuth(h.hostData.User, h.hostData.Password)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}

	return resp, nil
}

// SetHeader sets HTTP headers to use in requests
func (h *HTTP) SetHeader(key, value string) {
	h.headers.Set(key, value)
}

// SetHeaderDefault sets HTTP header as default
//
// Note: This will only set the header when the header is not already set.
func (h *HTTP) SetHeaderDefault(key, value string) {
	c := h.headers.Get(key)
	if c == "" {
		h.headers.Set(key, value)
	}
}

// SetMethod sets HTTP method to use in requests
func (h *HTTP) SetMethod(method string) {
	h.method = method
}

// GetURI gets the URI used in requests
func (h *HTTP) GetURI() string {
	return h.uri
}

// SetURI sets URI to use in requests
func (h *HTTP) SetURI(uri string) {
	h.uri = uri
}

// SetBody sets the body of the requests
func (h *HTTP) SetBody(body []byte) {
	h.body = body
}

// FetchContent makes an HTTP request to the configured url and returns the body content.
func (h *HTTP) FetchContent() ([]byte, error) {
	resp, err := h.FetchResponse()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d in %s: %s", resp.StatusCode, h.name, resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

// FetchScanner returns a Scanner for the content.
func (h *HTTP) FetchScanner() (*bufio.Scanner, error) {
	content, err := h.FetchContent()
	if err != nil {
		return nil, err
	}

	return bufio.NewScanner(bytes.NewReader(content)), nil
}

// FetchJSON makes an HTTP request to the configured url and returns the JSON content.
// This only works if the JSON output needed is in map[string]interface format.
func (h *HTTP) FetchJSON() (map[string]interface{}, error) {
	body, err := h.FetchContent()
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// getAuthHeaderFromToken reads a bearer authorizaiton token from the given file
func getAuthHeaderFromToken(path string) (string, error) {
	var token string

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "reading bearer token file")
	}

	if len(b) != 0 {
		if b[len(b)-1] == '\n' {
			b = b[0 : len(b)-1]
		}
		token = fmt.Sprintf("Bearer %s", string(b))
	}

	return token, nil
}
