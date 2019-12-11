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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type Connection struct {
	URL      string
	Username string
	Password string

	HTTP    *http.Client
	Version common.Version
}

type Client struct {
	Connection
}

func addToURL(_url, _path string, params url.Values) string {
	if len(params) == 0 {
		return _url + _path
	}

	return strings.Join([]string{_url, _path, "?", params.Encode()}, "")
}

func extractError(result []byte) error {
	var kibanaResult struct {
		Objects []struct {
			Error struct {
				Message string
			}
		}
	}
	if err := json.Unmarshal(result, &kibanaResult); err != nil {
		return errors.Wrap(err, "parsing kibana response")
	}
	for _, o := range kibanaResult.Objects {
		if o.Error.Message != "" {
			return errors.New(kibanaResult.Objects[0].Error.Message)
		}
	}
	return nil
}

// NewKibanaClient builds and returns a new Kibana client
func NewKibanaClient(cfg *common.Config) (*Client, error) {
	config := defaultClientConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return NewClientWithConfig(&config)
}

// NewClientWithConfig creates and returns a kibana client using the given config
func NewClientWithConfig(config *ClientConfig) (*Client, error) {
	p := config.Path
	if config.SpaceID != "" {
		p = path.Join(p, "s", config.SpaceID)
	}
	kibanaURL, err := common.MakeURL(config.Protocol, p, config.Host, 5601)
	if err != nil {
		return nil, fmt.Errorf("invalid Kibana host: %v", err)
	}

	u, err := url.Parse(kibanaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the Kibana URL: %v", err)
	}

	username := config.Username
	password := config.Password

	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		u.User = nil

		// Re-write URL without credentials.
		kibanaURL = u.String()
	}

	logp.Info("Kibana url: %s", kibanaURL)

	var dialer, tlsDialer transport.Dialer

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("fail to load the TLS config: %v", err)
	}

	dialer = transport.NetDialer(config.Timeout)
	tlsDialer, err = transport.TLSDialer(dialer, tlsConfig, config.Timeout)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Connection: Connection{
			URL:      kibanaURL,
			Username: username,
			Password: password,
			HTTP: &http.Client{
				Transport: &http.Transport{
					Dial:    dialer.Dial,
					DialTLS: tlsDialer.Dial,
				},
				Timeout: config.Timeout,
			},
		},
	}

	if !config.IgnoreVersion {
		if err = client.readVersion(); err != nil {
			return nil, fmt.Errorf("fail to get the Kibana version: %v", err)
		}
	}

	return client, nil
}

func (conn *Connection) Request(method, extraPath string,
	params url.Values, headers http.Header, body io.Reader) (int, []byte, error) {

	resp, err := conn.Send(method, extraPath, params, headers, body)
	if err != nil {
		return 0, nil, fmt.Errorf("fail to execute the HTTP %s request: %v", method, err)
	}
	defer resp.Body.Close()

	var retError error
	if resp.StatusCode >= 300 {
		retError = fmt.Errorf("%v", resp.Status)
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("fail to read response %s", err)
	}

	retError = extractError(result)
	return resp.StatusCode, result, retError
}

// Sends an application/json request to Kibana with appropriate kbn headers
func (conn *Connection) Send(method, extraPath string,
	params url.Values, headers http.Header, body io.Reader) (*http.Response, error) {

	reqURL := addToURL(conn.URL, extraPath, params)

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("fail to create the HTTP %s request: %+v", method, err)
	}

	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Set("kbn-xsrf", "1")

	for header, values := range headers {
		for _, value := range values {
			req.Header.Add(header, value)
		}
	}

	return conn.RoundTrip(req)
}

// Implements RoundTrip interface
func (conn *Connection) RoundTrip(r *http.Request) (*http.Response, error) {
	return conn.HTTP.Do(r)
}

func (client *Client) readVersion() error {
	type kibanaVersionResponse struct {
		Name    string `json:"name"`
		Version struct {
			Number   string `json:"number"`
			Snapshot bool   `json:"build_snapshot"`
		} `json:"version"`
	}

	type kibanaVersionResponse5x struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	code, result, err := client.Connection.Request("GET", "/api/status", nil, nil, nil)
	if err != nil || code >= 400 {
		return fmt.Errorf("HTTP GET request to %s/api/status fails: %v. Response: %s.",
			client.Connection.URL, err, truncateString(result))
	}

	var versionString string

	var kibanaVersion kibanaVersionResponse
	err = json.Unmarshal(result, &kibanaVersion)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the response from GET %s/api/status. Response: %s. Kibana status api returns: %v",
			client.Connection.URL, truncateString(result), err)
	}

	versionString = kibanaVersion.Version.Number

	if kibanaVersion.Version.Snapshot {
		// needed for the tests
		versionString += "-SNAPSHOT"
	}

	version, err := common.NewVersion(versionString)
	if err != nil {
		return fmt.Errorf("fail to parse kibana version (%v): %+v", versionString, err)
	}

	client.Version = *version
	return nil
}

// GetVersion returns the version read from kibana. The version is not set if
// IgnoreVersion was set when creating the client.
func (client *Client) GetVersion() common.Version { return client.Version }

func (client *Client) ImportJSON(url string, params url.Values, jsonBody map[string]interface{}) error {

	body, err := json.Marshal(jsonBody)
	if err != nil {
		logp.Err("Failed to json encode body (%v): %#v", err, jsonBody)
		return fmt.Errorf("fail to marshal the json content: %v", err)
	}

	statusCode, response, err := client.Connection.Request("POST", url, params, nil, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("%v. Response: %s", err, truncateString(response))
	}
	if statusCode >= 300 {
		return fmt.Errorf("returned %d to import file: %v. Response: %s", statusCode, err, response)
	}
	return nil
}

func (client *Client) Close() error { return nil }

// GetDashboard returns the dashboard with the given id with the index pattern removed
func (client *Client) GetDashboard(id string) (common.MapStr, error) {
	params := url.Values{}
	params.Add("dashboard", id)
	_, response, err := client.Request("GET", "/api/kibana/dashboards/export", params, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error exporting dashboard: %+v", err)
	}

	result, err := RemoveIndexPattern(response)
	if err != nil {
		return nil, fmt.Errorf("error removing index pattern: %+v", err)
	}

	return result, nil
}

// truncateString returns a truncated string if the length is greater than 250
// runes. If the string is truncated "... (truncated)" is appended. Newlines are
// replaced by spaces in the returned string.
//
// This function is useful for logging raw HTTP responses with errors when those
// responses can be very large (such as an HTML page with CSS content).
func truncateString(b []byte) string {
	const maxLength = 250
	runes := bytes.Runes(b)
	if len(runes) > maxLength {
		runes = append(runes[:maxLength], []rune("... (truncated)")...)
	}

	return strings.Replace(string(runes), "\n", " ", -1)
}
