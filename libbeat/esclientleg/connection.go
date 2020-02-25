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

package esclientleg

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common"
)

// Connection manages the connection for a given client.
type Connection struct {
	ConnectionSettings

	Encoder BodyEncoder

	// buffered bulk requests
	BulkRequ *BulkRequest

	version common.Version

	log *logp.Logger
}

// ConnectionSettings are the settings needed for a Connection
type ConnectionSettings struct {
	URL      string
	ProxyURL *url.URL

	Username string
	Password string
	APIKey   string
	Headers  map[string]string

	TLSConfig *tlscommon.TLSConfig

	HTTP *http.Client

	OnConnectCallback func() error

	Parameters       map[string]string
	CompressionLevel int
	EscapeHTML       bool
}

func NewConnection(settings ConnectionSettings) (*Connection, error) {
	var encoder BodyEncoder
	var err error
	compression := settings.CompressionLevel
	if compression == 0 {
		encoder = NewJSONEncoder(nil, settings.EscapeHTML)
	} else {
		encoder, err = NewGzipEncoder(compression, nil, settings.EscapeHTML)
		if err != nil {
			return nil, err
		}
	}

	bulkRequ, err := NewBulkRequest(settings.URL, "", "", settings.Parameters, nil)
	if err != nil {
		return nil, err
	}

	return &Connection{
		ConnectionSettings: settings,
		Encoder:            encoder,
		BulkRequ:           bulkRequ,
		log:                logp.NewLogger("esclientleg"),
	}, nil
}

// Connect connects the client. It runs a GET request against the root URL of
// the configured host, updates the known Elasticsearch version and calls
// globally configured handlers.
func (conn *Connection) Connect() error {
	versionString, err := conn.Ping()
	if err != nil {
		return err
	}

	if version, err := common.NewVersion(versionString); err != nil {
		conn.log.Errorf("Invalid version from Elasticsearch: %v", versionString)
		conn.version = common.Version{}
	} else {
		conn.version = *version
	}

	if conn.OnConnectCallback != nil {
		err = conn.OnConnectCallback()
		if err != nil {
			return fmt.Errorf("Connection marked as failed because the onConnect callback failed: %v", err)
		}
	}

	return nil
}

// Ping sends a GET request to the Elasticsearch.
func (conn *Connection) Ping() (string, error) {
	conn.log.Debugf("ES Ping(url=%v)", conn.URL)

	status, body, err := conn.execRequest("GET", conn.URL, nil)
	if err != nil {
		conn.log.Debugf("Ping request failed with: %v", err)
		return "", err
	}

	if status >= 300 {
		return "", fmt.Errorf("Non 2xx response code: %d", status)
	}

	var response struct {
		Version struct {
			Number string
		}
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("Failed to parse JSON response: %v", err)
	}

	conn.log.Debugf("Ping status code: %v", status)
	conn.log.Infof("Attempting to connect to Elasticsearch version %s", response.Version.Number)
	return response.Version.Number, nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	return nil
}

// Request sends a request via the connection.
func (conn *Connection) Request(
	method, path string,
	pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {

	url := addToURL(conn.URL, path, pipeline, params)
	conn.log.Debugf("%s %s %s %v", method, url, pipeline, body)

	return conn.RequestURL(method, url, body)
}

// RequestURL sends a request with the connection object to an alternative url
func (conn *Connection) RequestURL(
	method, url string,
	body interface{},
) (int, []byte, error) {

	if body == nil {
		return conn.execRequest(method, url, nil)
	}

	if err := conn.Encoder.Marshal(body); err != nil {
		conn.log.Warnf("Failed to json encode body (%v): %#v", err, body)
		return 0, nil, ErrJSONEncodeFailed
	}
	return conn.execRequest(method, url, conn.Encoder.Reader())
}

func (conn *Connection) execRequest(
	method, url string,
	body io.Reader,
) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		conn.log.Warnf("Failed to create request %+v", err)
		return 0, nil, err
	}
	if body != nil {
		conn.Encoder.AddHeader(&req.Header)
	}
	return conn.execHTTPRequest(req)
}

// GetVersion returns the elasticsearch version the client is connected to.
// The version is read and updated on 'Connect'.
func (conn *Connection) GetVersion() common.Version {
	return conn.version
}

// LoadJSON creates a PUT request based on a JSON document.
func (conn *Connection) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	status, body, err := conn.Request("PUT", path, "", nil, json)
	if err != nil {
		return body, fmt.Errorf("couldn't load json. Error: %s", err)
	}
	if status > 300 {
		return body, fmt.Errorf("couldn't load json. Status: %v", status)
	}

	return body, nil
}

func (conn *Connection) execHTTPRequest(req *http.Request) (int, []byte, error) {
	req.Header.Add("Accept", "application/json")

	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	if conn.APIKey != "" {
		req.Header.Add("Authorization", "ApiKey "+conn.APIKey)
	}

	for name, value := range conn.Headers {
		req.Header.Add(name, value)
	}

	// The stlib will override the value in the header based on the configured `Host`
	// on the request which default to the current machine.
	//
	// We use the normalized key header to retrieve the user configured value and assign it to the host.
	if host := req.Header.Get("Host"); host != "" {
		req.Host = host
	}

	resp, err := conn.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer closing(resp.Body, conn.log)

	status := resp.StatusCode
	obj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return status, nil, err
	}

	if status >= 300 {
		// add the response body with the error returned by Elasticsearch
		err = fmt.Errorf("%v: %s", resp.Status, obj)
	}

	return status, obj, err
}

func closing(c io.Closer, logger *logp.Logger) {
	err := c.Close()
	if err != nil {
		logger.Warn("Close failed with: %v", err)
	}
}
