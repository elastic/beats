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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.elastic.co/apm/module/apmelasticsearch/v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/productorigin"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/beats/v7/libbeat/version"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/testing"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/elastic-agent-libs/useragent"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

type esHTTPClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
	CloseIdleConnections()
}

// Connection manages the connection for a given client. Each connection is not-thread-safe and should not be shared
// between 2 different goroutines.
type Connection struct {
	ConnectionSettings

	Encoder BodyEncoder
	HTTP    esHTTPClient

	apiKeyAuthHeader string // Authorization HTTP request header with base64-encoded API key
	version          libversion.V
	log              *logp.Logger
	responseBuffer   *bytes.Buffer

	isServerless bool
}

// ConnectionSettings are the settings needed for a Connection
type ConnectionSettings struct {
	URL      string
	Beatname string

	Username string
	Password string
	APIKey   string // Raw API key, NOT base64-encoded
	Headers  map[string]string

	Kerberos *kerberos.Config

	OnConnectCallback func() error
	Observer          transport.IOStatser

	Parameters       map[string]string
	CompressionLevel int
	EscapeHTML       bool

	IdleConnTimeout time.Duration

	Transport httpcommon.HTTPTransportSettings
}

type ESPingData struct {
	Version ESVersionData `json:"version"`
	Name    string        `json:"name"`
}

type ESVersionData struct {
	Number      string `json:"number"`
	BuildFlavor string `json:"build_flavor"`
}

// NewConnection returns a new Elasticsearch client
func NewConnection(s ConnectionSettings) (*Connection, error) {
	logger := logp.NewLogger("esclientleg")

	if s.IdleConnTimeout == 0 {
		s.IdleConnTimeout = 1 * time.Minute
	}

	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elasticsearch URL: %w", err)
	}

	if u.User != nil {
		s.Username = u.User.Username()
		s.Password, _ = u.User.Password()
		u.User = nil

		// Re-write URL without credentials.
		s.URL = u.String()
	}
	logger.Infof("elasticsearch url: %s", s.URL)

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

	if s.Beatname == "" {
		s.Beatname = "Libbeat"
	}
	userAgent := useragent.UserAgent(s.Beatname, version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

	// Default the product origin header to beats if it wasn't already set.
	if _, ok := s.Headers[productorigin.Header]; !ok {
		if s.Headers == nil {
			s.Headers = make(map[string]string)
		}
		s.Headers[productorigin.Header] = productorigin.Beats
	}

	httpClient, err := s.Transport.Client(
		httpcommon.WithLogger(logger),
		httpcommon.WithIOStats(s.Observer),
		httpcommon.WithKeepaliveSettings{IdleConnTimeout: s.IdleConnTimeout},
		httpcommon.WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
			// when dropping the legacy client in favour of the official Go client, it should be instrumented
			// eg, like in https://github.com/elastic/apm-server/blob/7.7/elasticsearch/client.go
			return apmelasticsearch.WrapRoundTripper(rt)
		}),
		httpcommon.WithHeaderRoundTripper(map[string]string{"User-Agent": userAgent}),
	)
	if err != nil {
		return nil, err
	}

	esClient := esHTTPClient(httpClient)
	if s.Kerberos.IsEnabled() {
		esClient, err = kerberos.NewClient(s.Kerberos, httpClient, s.URL)
		if err != nil {
			return nil, err
		}
		logger.Info("kerberos client created")
	}

	conn := Connection{
		ConnectionSettings: s,
		HTTP:               esClient,
		Encoder:            encoder,
		log:                logger,
		responseBuffer:     bytes.NewBuffer(nil),
	}

	if s.APIKey != "" {
		conn.apiKeyAuthHeader = "ApiKey " + base64.StdEncoding.EncodeToString([]byte(s.APIKey))
	}

	return &conn, nil
}

// NewClients returns a list of Elasticsearch clients based on the given
// configuration. It accepts the same configuration parameters as the Elasticsearch
// output, except for the output specific configuration options.  If multiple hosts
// are defined in the configuration, a client is returned for each of them.
// The returned Connection is a non-thread-safe connection.
func NewClients(cfg *cfg.C, beatname string) ([]Connection, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if proxyURL := config.Transport.Proxy.URL; proxyURL != nil {
		logp.Debug("breaking down proxy URL. Scheme: '%s', host[:port]: '%s', path: '%s'", proxyURL.Scheme, proxyURL.Host, proxyURL.Path)
		logp.Info("using proxy URL: %s", proxyURL.URI().String())
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
			Beatname:         beatname,
			Kerberos:         config.Kerberos,
			Username:         config.Username,
			Password:         config.Password,
			APIKey:           config.APIKey,
			Parameters:       params,
			Headers:          config.Headers,
			CompressionLevel: config.CompressionLevel,
			Transport:        config.Transport,
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

// NewConnectedClient returns a non-thread-safe connection. Make sure for each goroutine you initialize a new connection.
func NewConnectedClient(cfg *cfg.C, beatname string) (*Connection, error) {
	clients, err := NewClients(cfg, beatname)
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
	if conn.log == nil {
		conn.log = logp.NewLogger("esclientleg")
	}
	if err := conn.getVersion(); err != nil {
		return err
	}

	if conn.OnConnectCallback != nil {
		if err := conn.OnConnectCallback(); err != nil {
			return fmt.Errorf("Connection marked as failed because the onConnect callback failed: %w", err)
		}
	}

	return nil
}

// Ping sends a GET request to the Elasticsearch.
func (conn *Connection) Ping() (ESPingData, error) {
	conn.log.Debugf("ES Ping(url=%v)", conn.URL)

	status, body, err := conn.execRequest("GET", conn.URL, nil)
	if err != nil {
		conn.log.Debugf("Ping request failed with: %v", err)
		return ESPingData{}, err
	}

	if status >= 300 {
		return ESPingData{}, fmt.Errorf("non 2xx response code: %d", status)
	}

	response := ESPingData{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return ESPingData{}, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	conn.log.Debugf("Ping status code: %v", status)
	conn.log.Infof("Attempting to connect to Elasticsearch version %s (%s)", response.Version.Number, response.Version.BuildFlavor)
	return response, nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	conn.HTTP.CloseIdleConnections()
	return nil
}

func (conn *Connection) Test(d testing.Driver) {
	d.Run("elasticsearch: "+conn.URL, func(d testing.Driver) {
		u, err := url.Parse(conn.URL)
		d.Fatal("parse url", err)

		address := u.Host

		d.Run("connection", func(d testing.Driver) {
			netDialer := transport.TestNetDialer(d, conn.Transport.Timeout)
			_, err = netDialer.Dial("tcp", address)
			d.Fatal("dial up", err)
		})

		if u.Scheme != "https" {
			d.Warn("TLS", "secure connection disabled")
		} else {
			d.Run("TLS", func(d testing.Driver) {
				tls, err := tlscommon.LoadTLSConfig(conn.Transport.TLS)
				if err != nil {
					d.Fatal("load tls config", err)
				}

				netDialer := transport.NetDialer(conn.Transport.Timeout)
				tlsDialer := transport.TestTLSDialer(d, netDialer, tls, conn.Transport.Timeout)
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
	req, err := http.NewRequest(method, url, body) //nolint:noctx // keep legacy behaviour
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
func (conn *Connection) GetVersion() libversion.V {
	if !conn.version.IsValid() {
		_ = conn.getVersion()
	}

	return conn.version
}

// IsServerless returns true if we're connected to a serverless ES instance
func (conn *Connection) IsServerless() bool {
	// make sure we've initialized the version state first
	_ = conn.GetVersion()
	return conn.isServerless
}

func (conn *Connection) getVersion() error {
	versionData, err := conn.Ping()
	if err != nil {
		return err
	}

	if v, err := libversion.New(versionData.Version.Number); err != nil {
		conn.log.Errorf("Invalid version from Elasticsearch: %v", versionData.Version.Number)
		conn.version = libversion.V{}
	} else {
		conn.version = *v
	}

	if versionData.Version.BuildFlavor == "serverless" {
		conn.log.Info("build flavor of es is serverless, marking connection as serverless")
		conn.isServerless = true
	} else if versionData.Version.BuildFlavor == "default" {
		conn.isServerless = false
		// not sure if this is even possible, just being defensive
	} else {
		conn.log.Infof("Got unexpected build flavor '%s'", versionData.Version.BuildFlavor)
	}

	return nil
}

// LoadJSON creates a PUT request based on a JSON document.
func (conn *Connection) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	status, body, err := conn.Request("PUT", path, "", nil, json)
	if err != nil {
		return body, fmt.Errorf("couldn't load json. Error: %w", err)
	}
	if status > 300 {
		return body, fmt.Errorf("couldn't load json. Status: %v", status)
	}

	return body, nil
}

// execHTTPRequest executes the http request and consumes the response in a non-thread-safe way.
// The return is a triple of status code, response as byte array, error if the request produced any error.
func (conn *Connection) execHTTPRequest(req *http.Request) (int, []byte, error) {
	req.Header.Add("Accept", "application/json")

	if conn.Username != "" || conn.Password != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	if conn.apiKeyAuthHeader != "" {
		req.Header.Add("Authorization", conn.apiKeyAuthHeader)
	}

	for name, value := range conn.Headers {
		if name == "Content-Type" || name == "Accept" {
			req.Header.Set(name, value)
		} else {
			req.Header.Add(name, value)
		}
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
	conn.responseBuffer.Reset()
	_, err = io.Copy(conn.responseBuffer, resp.Body)
	if err != nil {
		return status, nil, err
	}

	if status >= 300 {
		// add the response body with the error returned by Elasticsearch
		err = fmt.Errorf("%v: %s", resp.Status, conn.responseBuffer.Bytes())
	}

	return status, conn.responseBuffer.Bytes(), err
}

func closing(c io.Closer, logger *logp.Logger) {
	err := c.Close()
	if err != nil {
		logger.Warn("Close failed with: %v", err)
	}
}
