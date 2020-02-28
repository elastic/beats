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

package eslegclient

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/testing"
)

// Connection manages the connection for a given client.
type Connection struct {
	ConnectionSettings

	Encoder BodyEncoder

	// buffered bulk requests
	BulkRequ *BulkRequest

	HTTP *http.Client

	version common.Version

	log *logp.Logger
}

// ConnectionSettings are the settings needed for a Connection
type ConnectionSettings struct {
	URL          string
	Proxy        *url.URL
	ProxyDisable bool

	Username string
	Password string
	APIKey   string
	Headers  map[string]string

	TLS *tlscommon.TLSConfig

	OnConnectCallback func() error
	Observer          transport.IOStatser

	Parameters       map[string]string
	CompressionLevel int
	EscapeHTML       bool
	Timeout          time.Duration
}

// NewConnection returns a new Elasticsearch client
func NewConnection(s ConnectionSettings) (*Connection, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elasticsearch URL: %v", err)
	}

	if u.User != nil {
		s.Username = u.User.Username()
		s.Password, _ = u.User.Password()
		u.User = nil

		// Re-write URL without credentials.
		s.URL = u.String()
	}
	logp.Info("elasticsearch url: %s", s.URL)

	// TODO: add socks5 proxy support
	var dialer, tlsDialer transport.Dialer

	dialer = transport.NetDialer(s.Timeout)
	tlsDialer, err = transport.TLSDialer(dialer, s.TLS, s.Timeout)
	if err != nil {
		return nil, err
	}

	if st := s.Observer; st != nil {
		dialer = transport.StatsDialer(dialer, st)
		tlsDialer = transport.StatsDialer(tlsDialer, st)
	}

	var encoder BodyEncoder
	compression := s.CompressionLevel
	if compression == 0 {
		encoder = NewJSONEncoder(nil, s.EscapeHTML)
	} else {
		encoder, err = NewGzipEncoder(compression, nil, s.EscapeHTML)
		if err != nil {
			return nil, err
		}
	}

	bulkRequ, err := NewBulkRequest(s.URL, "", "", s.Parameters, nil)
	if err != nil {
		return nil, err
	}

	var proxy func(*http.Request) (*url.URL, error)
	if !s.ProxyDisable {
		proxy = http.ProxyFromEnvironment
		if s.Proxy != nil {
			proxy = http.ProxyURL(s.Proxy)
		}
	}

	return &Connection{
		ConnectionSettings: s,
		HTTP: &http.Client{
			Transport: &http.Transport{
				Dial:            dialer.Dial,
				DialTLS:         tlsDialer.Dial,
				TLSClientConfig: s.TLS.ToConfig(),
				Proxy:           proxy,
			},
			Timeout: s.Timeout,
		},
		Encoder:  encoder,
		BulkRequ: bulkRequ,
		log:      logp.NewLogger("esclientleg"),
	}, nil
}

// NewClients returns a list of Elasticsearch clients based on the given
// configuration. It accepts the same configuration parameters as the Elasticsearch
// output, except for the output specific configuration options.  If multiple hosts
// are defined in the configuration, a client is returned for each of them.
func NewClients(cfg *common.Config) ([]Connection, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	var proxyURL *url.URL
	if !config.ProxyDisable {
		proxyURL, err = common.ParseURL(config.ProxyURL)
		if err != nil {
			return nil, err
		}
		if proxyURL != nil {
			logp.Info("using proxy URL: %s", proxyURL)
		}
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := []Connection{}
	for _, host := range config.Hosts {
		esURL, err := common.MakeURL(config.Protocol, config.Path, host, 9200)
		if err != nil {
			logp.Err("invalid host param set: %s, Error: %v", host, err)
			return nil, err
		}

		client, err := NewConnection(ConnectionSettings{
			URL:              esURL,
			Proxy:            proxyURL,
			ProxyDisable:     config.ProxyDisable,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			APIKey:           config.APIKey,
			Parameters:       params,
			Headers:          config.Headers,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		})
		if err != nil {
			return clients, err
		}
		clients = append(clients, *client)
	}
	if len(clients) == 0 {
		return clients, fmt.Errorf("no hosts defined in the config")
	}
	return clients, nil
}

func NewConnectedClient(cfg *common.Config) (*Connection, error) {
	clients, err := NewClients(cfg)
	if err != nil {
		return nil, err
	}

	errors := []string{}

	for _, client := range clients {
		err = client.Connect()
		if err != nil {
			const errMsg = "error connecting to Elasticsearch at %v: %v"
			client.log.Errorf(errMsg, client.URL, err)
			err = fmt.Errorf(errMsg, client.URL, err)
			errors = append(errors, err.Error())
			continue
		}
		return &client, nil
	}
	return nil, fmt.Errorf("couldn't connect to any of the configured Elasticsearch hosts. Errors: %v", errors)
}

// Connect connects the client. It runs a GET request against the root URL of
// the configured host, updates the known Elasticsearch version and calls
// globally configured handlers.
func (conn *Connection) Connect() error {
	if err := conn.getVersion(); err != nil {
		return err
	}

	if conn.OnConnectCallback != nil {
		if err := conn.OnConnectCallback(); err != nil {
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

func (conn *Connection) Test(d testing.Driver) {
	d.Run("elasticsearch: "+conn.URL, func(d testing.Driver) {
		u, err := url.Parse(conn.URL)
		d.Fatal("parse url", err)

		address := u.Host

		d.Run("connection", func(d testing.Driver) {
			netDialer := transport.TestNetDialer(d, conn.Timeout)
			_, err = netDialer.Dial("tcp", address)
			d.Fatal("dial up", err)
		})

		if u.Scheme != "https" {
			d.Warn("TLS", "secure connection disabled")
		} else {
			d.Run("TLS", func(d testing.Driver) {
				netDialer := transport.NetDialer(conn.Timeout)
				tlsDialer, err := transport.TestTLSDialer(d, netDialer, conn.TLS, conn.Timeout)
				_, err = tlsDialer.Dial("tcp", address)
				d.Fatal("dial up", err)
			})
		}

		err = conn.Connect()
		d.Fatal("talk to server", err)
		version := conn.GetVersion()
		d.Info("version", version.String())
	})
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
func (conn *Connection) GetVersion() common.Version {
	if !conn.version.IsValid() {
		conn.getVersion()
	}

	return conn.version
}

func (conn *Connection) getVersion() error {
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

	return nil
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
