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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// We started using Saved Objects API in 7.15. But to help integration
	// developers migrate their dashboards we are more lenient.
	MinimumRequiredVersionSavedObjects = common.MustNewVersion("7.14.0")
)

type Connection struct {
	URL          string
	Username     string
	Password     string
	APIKey       string
	ServiceToken string
	Headers      http.Header

	HTTP    *http.Client
	Version common.Version
}

type Client struct {
	Connection
	log *logp.Logger
}

func addToURL(_url, _path string, params url.Values) string {
	if len(params) == 0 {
		return _url + _path
	}

	return strings.Join([]string{_url, _path, "?", params.Encode()}, "")
}

func extractError(result []byte) error {
	var kibanaResult struct {
		Message    string
		Attributes struct {
			Objects []struct {
				Id    string
				Error struct {
					Message string
				}
			}
		}
	}
	if err := json.Unmarshal(result, &kibanaResult); err != nil {
		return err
	}
	var errs multierror.Errors
	if kibanaResult.Message != "" {
		for _, err := range kibanaResult.Attributes.Objects {
			errs = append(errs, fmt.Errorf("id: %s, message: %s", err.Id, err.Error.Message))
		}
		return fmt.Errorf("%s: %+v", kibanaResult.Message, errs.Err())
	}
	return nil
}

func extractMessage(result []byte) error {
	var kibanaResult struct {
		Success bool
		Errors  []struct {
			Id    string
			Type  string
			Error struct {
				Type       string
				References []struct {
					Type string
					Id   string
				}
			}
		}
	}
	if err := json.Unmarshal(result, &kibanaResult); err != nil {
		return nil
	}

	if !kibanaResult.Success {
		var errs multierror.Errors
		for _, err := range kibanaResult.Errors {
			errs = append(errs, fmt.Errorf("error: %s, asset ID=%s; asset type=%s; references=%+v", err.Error.Type, err.Id, err.Type, err.Error.References))
		}
		return errs.Err()
	}

	return nil
}

// NewKibanaClient builds and returns a new Kibana client
func NewKibanaClient(cfg *config.C, beatname string) (*Client, error) {
	config := DefaultClientConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return NewClientWithConfig(&config, beatname)
}

// NewClientWithConfig creates and returns a kibana client using the given config
func NewClientWithConfig(config *ClientConfig, beatname string) (*Client, error) {
	return NewClientWithConfigDefault(config, 5601, beatname)
}

// NewClientWithConfig creates and returns a kibana client using the given config
func NewClientWithConfigDefault(config *ClientConfig, defaultPort int, beatname string) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	p := config.Path
	if config.SpaceID != "" {
		p = path.Join(p, "s", config.SpaceID)
	}
	kibanaURL, err := common.MakeURL(config.Protocol, p, config.Host, defaultPort)
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

		if config.APIKey != "" && (username != "" || password != "") {
			return nil, fmt.Errorf("cannot set api_key with username/password in Kibana URL")
		}

		// Re-write URL without credentials.
		kibanaURL = u.String()
	}

	log := logp.NewLogger("kibana")
	log.Infof("Kibana url: %s", kibanaURL)

	headers := make(http.Header)
	for k, v := range config.Headers {
		headers.Set(k, v)
	}

	if beatname == "" {
		beatname = "Libbeat"
	}
	userAgent := useragent.UserAgent(beatname)
	rt, err := config.Transport.Client(httpcommon.WithHeaderRoundTripper(map[string]string{"User-Agent": userAgent}))
	if err != nil {
		return nil, err
	}

	client := &Client{
		Connection: Connection{
			URL:          kibanaURL,
			Username:     username,
			Password:     password,
			APIKey:       config.APIKey,
			ServiceToken: config.ServiceToken,
			Headers:      headers,
			HTTP:         rt,
		},
		log: log,
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

	if resp.StatusCode >= 300 {
		retError = extractError(result)
	} else {
		retError = extractMessage(result)
	}
	return resp.StatusCode, result, retError
}

// Sends an application/json request to Kibana with appropriate kbn headers
func (conn *Connection) Send(method, extraPath string,
	params url.Values, headers http.Header, body io.Reader) (*http.Response, error) {

	return conn.SendWithContext(context.Background(), method, extraPath, params, headers, body)
}

// SendWithContext sends an application/json request to Kibana with appropriate kbn headers and the given context.
func (conn *Connection) SendWithContext(ctx context.Context, method, extraPath string,
	params url.Values, headers http.Header, body io.Reader) (*http.Response, error) {

	reqURL := addToURL(conn.URL, extraPath, params)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("fail to create the HTTP %s request: %+v", method, err)
	}

	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}
	if conn.APIKey != "" {
		v := "ApiKey " + base64.StdEncoding.EncodeToString([]byte(conn.APIKey))
		req.Header.Set("Authorization", v)
	}
	if conn.ServiceToken != "" {
		v := "Bearer " + conn.ServiceToken
		req.Header.Set("Authorization", v)
	}

	addHeaders(req.Header, conn.Headers)
	addHeaders(req.Header, headers)

	contentType := req.Header.Get("Content-Type")
	contentType, _, _ = mime.ParseMediaType(contentType)
	if contentType != "multipart/form-data" && contentType != "application/ndjson" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("kbn-xsrf", "1")

	return conn.RoundTrip(req)
}

func addHeaders(out, in http.Header) {
	for k, vs := range in {
		for _, v := range vs {
			out.Add(k, v)
		}
	}
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

func (client *Client) ImportMultiPartFormFile(url string, params url.Values, filename string, contents string) error {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	pHeaders := textproto.MIMEHeader{}
	pHeaders.Add("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	pHeaders.Add("Content-Type", "application/ndjson")

	p, err := w.CreatePart(pHeaders)
	if err != nil {
		return fmt.Errorf("failed to create multipart writer for payload: %+v", err)
	}
	_, err = io.Copy(p, strings.NewReader(contents))
	if err != nil {
		return fmt.Errorf("failed to copy contents of the object: %+v", err)
	}
	w.Close()

	headers := http.Header{}
	headers.Add("Content-Type", w.FormDataContentType())
	statusCode, response, err := client.Connection.Request("POST", url, params, headers, buf)
	if err != nil || statusCode >= 300 {
		return fmt.Errorf("returned %d to import file: %v. Response: %s", statusCode, err, response)
	}

	client.log.Debugf("Imported multipart file to %s with params %v", url, params)
	return nil
}

func (client *Client) Close() error { return nil }

// GetDashboard returns the dashboard with the given id with the index pattern removed
func (client *Client) GetDashboard(id string) ([]byte, error) {
	if client.Version.LessThan(MinimumRequiredVersionSavedObjects) {
		return nil, fmt.Errorf("Kibana version must be at least " + MinimumRequiredVersionSavedObjects.String())
	}

	body := fmt.Sprintf(`{"objects": [{"type": "dashboard", "id": "%s" }], "includeReferencesDeep": true, "excludeExportDetails": true}`, id)
	statusCode, response, err := client.Request("POST", "/api/saved_objects/_export", nil, nil, strings.NewReader(body))
	if err != nil || statusCode >= 300 {
		return nil, fmt.Errorf("error exporting dashboard: %+v, code: %d", err, statusCode)
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
