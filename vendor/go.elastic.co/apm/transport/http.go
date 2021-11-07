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

package transport // import "go.elastic.co/apm/transport"

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/apmconfig"
	"go.elastic.co/apm/internal/apmversion"
	"go.elastic.co/apm/internal/configutil"
)

const (
	intakePath  = "/intake/v2/events"
	profilePath = "/intake/v2/profile"
	configPath  = "/config/v1/agents"

	envAPIKey           = "ELASTIC_APM_API_KEY"
	envSecretToken      = "ELASTIC_APM_SECRET_TOKEN"
	envServerURLs       = "ELASTIC_APM_SERVER_URLS"
	envServerURL        = "ELASTIC_APM_SERVER_URL"
	envServerTimeout    = "ELASTIC_APM_SERVER_TIMEOUT"
	envServerCert       = "ELASTIC_APM_SERVER_CERT"
	envVerifyServerCert = "ELASTIC_APM_VERIFY_SERVER_CERT"
)

var (
	// Take a copy of the http.DefaultTransport pointer,
	// in case another package replaces the value later.
	defaultHTTPTransport = http.DefaultTransport.(*http.Transport)

	defaultServerURL, _  = url.Parse("http://localhost:8200")
	defaultServerTimeout = 30 * time.Second
)

// HTTPTransport is an implementation of Transport, sending payloads via
// a net/http client.
type HTTPTransport struct {
	// Client exposes the http.Client used by the HTTPTransport for
	// sending requests to the APM Server.
	Client         *http.Client
	intakeHeaders  http.Header
	configHeaders  http.Header
	profileHeaders http.Header
	shuffleRand    *rand.Rand

	urlIndex    int32
	intakeURLs  []*url.URL
	configURLs  []*url.URL
	profileURLs []*url.URL
}

// NewHTTPTransport returns a new HTTPTransport which can be used for
// streaming data to the APM Server. The returned HTTPTransport will be
// initialized using the following environment variables:
//
// - ELASTIC_APM_SERVER_URL: the APM Server URL used for sending
//   requests. If no URL is specified, then the transport will use the
//   default URL "http://localhost:8200".
//
// - ELASTIC_APM_SERVER_TIMEOUT: timeout for requests to the APM Server.
//   If not specified, defaults to 30 seconds.
//
// - ELASTIC_APM_SECRET_TOKEN: used to authenticate the agent.
//
// - ELASTIC_APM_SERVER_CERT: path to a PEM-encoded certificate that
//   must match the APM Server-supplied certificate. This can be used
//   to pin a self signed certificate. If this is set, then
//   ELASTIC_APM_VERIFY_SERVER_CERT is ignored.
//
// - ELASTIC_APM_VERIFY_SERVER_CERT: if set to "false", the transport
//   will not verify the APM Server's TLS certificate. Only relevant
//   when using HTTPS. By default, the transport will verify server
//   certificates.
//
func NewHTTPTransport() (*HTTPTransport, error) {
	verifyServerCert, err := configutil.ParseBoolEnv(envVerifyServerCert, true)
	if err != nil {
		return nil, err
	}

	serverTimeout, err := configutil.ParseDurationEnv(envServerTimeout, defaultServerTimeout)
	if err != nil {
		return nil, err
	}
	if serverTimeout < 0 {
		serverTimeout = 0
	}

	serverURLs, err := initServerURLs()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: !verifyServerCert}
	serverCertPath := os.Getenv(envServerCert)
	if serverCertPath != "" {
		serverCert, err := loadCertificate(serverCertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load certificate from %s", serverCertPath)
		}
		// Disable standard verification, we'll check that the
		// server supplies the exact certificate provided.
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return verifyPeerCertificate(rawCerts, serverCert)
		}
	}

	client := &http.Client{
		Timeout: serverTimeout,
		Transport: &http.Transport{
			Proxy:                 defaultHTTPTransport.Proxy,
			DialContext:           defaultHTTPTransport.DialContext,
			MaxIdleConns:          defaultHTTPTransport.MaxIdleConns,
			IdleConnTimeout:       defaultHTTPTransport.IdleConnTimeout,
			TLSHandshakeTimeout:   defaultHTTPTransport.TLSHandshakeTimeout,
			ExpectContinueTimeout: defaultHTTPTransport.ExpectContinueTimeout,
			TLSClientConfig:       tlsConfig,
		},
	}

	commonHeaders := make(http.Header)
	commonHeaders.Set("User-Agent", defaultUserAgent())

	intakeHeaders := copyHeaders(commonHeaders)
	intakeHeaders.Set("Content-Type", "application/x-ndjson")
	intakeHeaders.Set("Content-Encoding", "deflate")
	intakeHeaders.Set("Transfer-Encoding", "chunked")

	profileHeaders := copyHeaders(commonHeaders)

	t := &HTTPTransport{
		Client:         client,
		configHeaders:  commonHeaders,
		intakeHeaders:  intakeHeaders,
		profileHeaders: profileHeaders,
	}
	if apiKey := os.Getenv(envAPIKey); apiKey != "" {
		t.SetAPIKey(apiKey)
	} else if secretToken := os.Getenv(envSecretToken); secretToken != "" {
		t.SetSecretToken(secretToken)
	}
	t.SetServerURL(serverURLs...)
	return t, nil
}

// SetServerURL sets the APM Server URL (or URLs) for sending requests.
// At least one URL must be specified, or the method will panic. The
// list will be randomly shuffled.
func (t *HTTPTransport) SetServerURL(u ...*url.URL) {
	if len(u) == 0 {
		panic("SetServerURL expects at least one URL")
	}
	intakeURLs := make([]*url.URL, len(u))
	configURLs := make([]*url.URL, len(u))
	profileURLs := make([]*url.URL, len(u))
	for i, u := range u {
		intakeURLs[i] = urlWithPath(u, intakePath)
		configURLs[i] = urlWithPath(u, configPath)
		profileURLs[i] = urlWithPath(u, profilePath)
	}
	if n := len(intakeURLs); n > 0 {
		if t.shuffleRand == nil {
			t.shuffleRand = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
		for i := n - 1; i > 0; i-- {
			j := t.shuffleRand.Intn(i + 1)
			intakeURLs[i], intakeURLs[j] = intakeURLs[j], intakeURLs[i]
			configURLs[i], configURLs[j] = configURLs[j], configURLs[i]
			profileURLs[i], profileURLs[j] = profileURLs[j], profileURLs[i]
		}
	}
	t.intakeURLs = intakeURLs
	t.configURLs = configURLs
	t.profileURLs = profileURLs
	t.urlIndex = 0
}

// SetUserAgent sets the User-Agent header that will be sent with each request.
func (t *HTTPTransport) SetUserAgent(ua string) {
	t.setCommonHeader("User-Agent", ua)
}

// SetSecretToken sets the Authorization header with the given secret token.
//
// This overrides the value specified via the ELASTIC_APM_SECRET_TOKEN or
// ELASTIC_APM_API_KEY environment variables, if either are set.
func (t *HTTPTransport) SetSecretToken(secretToken string) {
	if secretToken != "" {
		t.setCommonHeader("Authorization", "Bearer "+secretToken)
	} else {
		t.deleteCommonHeader("Authorization")
	}
}

// SetAPIKey sets the Authorization header with the given API Key.
//
// This overrides the value specified via the ELASTIC_APM_SECRET_TOKEN or
// ELASTIC_APM_API_KEY environment variables, if either are set.
func (t *HTTPTransport) SetAPIKey(apiKey string) {
	if apiKey != "" {
		t.setCommonHeader("Authorization", "ApiKey "+apiKey)
	} else {
		t.deleteCommonHeader("Authorization")
	}
}

func (t *HTTPTransport) setCommonHeader(key, value string) {
	t.configHeaders.Set(key, value)
	t.intakeHeaders.Set(key, value)
	t.profileHeaders.Set(key, value)
}

func (t *HTTPTransport) deleteCommonHeader(key string) {
	t.configHeaders.Del(key)
	t.intakeHeaders.Del(key)
	t.profileHeaders.Del(key)
}

// SendStream sends the stream over HTTP. If SendStream returns an error and
// the transport is configured with more than one APM Server URL, then the
// following request will be sent to the next URL in the list.
func (t *HTTPTransport) SendStream(ctx context.Context, r io.Reader) error {
	urlIndex := atomic.LoadInt32(&t.urlIndex)
	intakeURL := t.intakeURLs[urlIndex]
	req := t.newRequest("POST", intakeURL)
	req = requestWithContext(ctx, req)
	req.Header = t.intakeHeaders
	req.Body = ioutil.NopCloser(r)
	if err := t.sendStreamRequest(req); err != nil {
		atomic.StoreInt32(&t.urlIndex, (urlIndex+1)%int32(len(t.intakeURLs)))
		return err
	}
	return nil
}

func (t *HTTPTransport) sendStreamRequest(req *http.Request) error {
	resp, err := t.Client.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending event request failed")
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		resp.Body.Close()
		return nil
	}
	defer resp.Body.Close()

	result := newHTTPError(resp)
	if resp.StatusCode == http.StatusNotFound && result.Message == "404 page not found" {
		// This may be an old (pre-6.5) APM server
		// that does not support the v2 intake API.
		result.Message = fmt.Sprintf("%s not found (requires APM Server 6.5.0 or newer)", req.URL)
	}
	return result
}

// SendProfile sends a symbolised pprof profile, encoded as protobuf, and gzip-compressed.
//
// NOTE this is an experimental API, and may be removed in a future minor version, without
// being considered a breaking change.
func (t *HTTPTransport) SendProfile(
	ctx context.Context,
	metadataReader io.Reader,
	profileReaders ...io.Reader,
) error {
	urlIndex := atomic.LoadInt32(&t.urlIndex)
	profileURL := t.profileURLs[urlIndex]
	req := t.newRequest("POST", profileURL)
	req = requestWithContext(ctx, req)
	req.Header = t.profileHeaders

	writeBody := func(w *multipart.Writer) error {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="metadata"`))
		h.Set("Content-Type", "application/json")
		part, err := w.CreatePart(h)
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, metadataReader); err != nil {
			return err
		}

		for _, profileReader := range profileReaders {
			h = make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="profile"`))
			h.Set("Content-Type", `application/x-protobuf; messageType="perftools.profiles.Profile"`)
			part, err = w.CreatePart(h)
			if err != nil {
				return err
			}
			if _, err := io.Copy(part, profileReader); err != nil {
				return err
			}
		}
		return w.Close()
	}
	pipeR, pipeW := io.Pipe()
	mpw := multipart.NewWriter(pipeW)
	req.Header.Set("Content-Type", mpw.FormDataContentType())
	req.Body = pipeR
	go func() {
		err := writeBody(mpw)
		pipeW.CloseWithError(err)
	}()
	return t.sendProfileRequest(req)
}

func (t *HTTPTransport) sendProfileRequest(req *http.Request) error {
	resp, err := t.Client.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending profile request failed")
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		resp.Body.Close()
		return nil
	}
	defer resp.Body.Close()

	result := newHTTPError(resp)
	if resp.StatusCode == http.StatusNotFound && result.Message == "404 page not found" {
		// TODO(axw) correct minimum server version.
		result.Message = fmt.Sprintf("%s not found (requires APM Server 7.5.0 or newer)", req.URL)
	}
	return result
}

// WatchConfig polls the APM Server for agent config changes, sending
// them over the returned channel.
func (t *HTTPTransport) WatchConfig(ctx context.Context, args apmconfig.WatchParams) <-chan apmconfig.Change {
	// We have an initial delay to allow application initialisation code
	// to close apm.DefaultTracer, which would cancel watching config.
	const initialDelay = 1 * time.Second

	changes := make(chan apmconfig.Change)
	go func() {
		defer close(changes)

		var etag string
		var out chan apmconfig.Change
		var change apmconfig.Change
		timer := time.NewTimer(initialDelay)
		for {
			select {
			case <-ctx.Done():
				return
			case out <- change:
				out = nil
				change = apmconfig.Change{}
				continue
			case <-timer.C:
			}

			urlIndex := atomic.LoadInt32(&t.urlIndex)
			query := make(url.Values)
			query.Set("service.name", args.Service.Name)
			if args.Service.Environment != "" {
				query.Set("service.environment", args.Service.Environment)
			}
			url := *t.configURLs[urlIndex]
			url.RawQuery = query.Encode()

			req := t.newRequest("GET", &url)
			req.Header = t.configHeaders
			if etag != "" {
				req.Header = copyHeaders(req.Header)
				req.Header.Set("If-None-Match", strconv.QuoteToASCII(etag))
			}

			req = requestWithContext(ctx, req)
			resp := t.configRequest(req)
			var send bool
			if resp.err != nil {
				// The request will have failed if the context has been
				// cancelled. No need to send a a change in this case.
				send = ctx.Err() == nil
			}
			if !send && resp.attrs != nil {
				etag = resp.etag
				send = true
			}
			if send {
				change = apmconfig.Change{Err: resp.err, Attrs: resp.attrs}
				out = changes
			}
			timer.Reset(resp.maxAge)
		}
	}()
	return changes
}

func (t *HTTPTransport) configRequest(req *http.Request) configResponse {
	// defaultMaxAge is the default amount of time to wait between
	// requests. This should only be used when the server does not
	// respond with a Cache-Control header, or where the header is
	// malformed.
	const defaultMaxAge = 5 * time.Minute

	resp, err := t.Client.Do(req)
	if err != nil {
		// TODO(axw) this might indicate that the APM Server is unavailable.
		// In this case, we should allow a change in URL due to SendStream
		// to cut the defaultMaxAge delay short.
		return configResponse{
			err:    errors.Wrap(err, "sending config request failed"),
			maxAge: defaultMaxAge,
		}
	}
	defer resp.Body.Close()

	var response configResponse
	if etag, err := strconv.Unquote(resp.Header.Get("Etag")); err == nil {
		response.etag = etag
	}
	cacheControl := parseCacheControl(resp.Header.Get("Cache-Control"))
	response.maxAge = cacheControl.maxAge
	if response.maxAge < 0 {
		response.maxAge = defaultMaxAge
	}

	switch resp.StatusCode {
	case http.StatusNotModified, http.StatusForbidden, http.StatusNotFound:
		// 304 (Not Modified) is returned when the config has not changed since the previous query.
		// 403 (Forbidden) is returned if the server does not have the connection to Kibana enabled.
		// 404 (Not Found) is returned by old servers that do not implement the config endpoint.
		return response
	case http.StatusOK:
		attrs := make(map[string]string)
		// TODO(axw) handling EOF shouldn't be necessary, server currently responds with an empty
		// body when there is no config.
		if err := json.NewDecoder(resp.Body).Decode(&attrs); err != nil && err != io.EOF {
			response.err = err
		} else {
			response.attrs = attrs
		}
		return response
	}
	response.err = newHTTPError(resp)
	return response
}

func (t *HTTPTransport) newRequest(method string, url *url.URL) *http.Request {
	req := &http.Request{
		Method:     method,
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       url.Host,
	}
	return req
}

func urlWithPath(url *url.URL, p string) *url.URL {
	urlCopy := *url
	urlCopy.Path = path.Clean(urlCopy.Path + p)
	if urlCopy.RawPath != "" {
		urlCopy.RawPath = path.Clean(urlCopy.RawPath + p)
	}
	return &urlCopy
}

// HTTPError is an error returned by HTTPTransport methods when requests fail.
type HTTPError struct {
	Response *http.Response
	Message  string
}

func newHTTPError(resp *http.Response) *HTTPError {
	bodyContents, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		resp.Body = ioutil.NopCloser(bytes.NewReader(bodyContents))
	}
	return &HTTPError{
		Response: resp,
		Message:  strings.TrimSpace(string(bodyContents)),
	}
}

func (e *HTTPError) Error() string {
	msg := fmt.Sprintf("request failed with %s", e.Response.Status)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

// initServerURLs parses ELASTIC_APM_SERVER_URLS if specified,
// otherwise parses ELASTIC_APM_SERVER_URL if specified. If
// neither are specified, then the default localhost URL is
// returned.
func initServerURLs() ([]*url.URL, error) {
	key := envServerURLs
	value := os.Getenv(key)
	if value == "" {
		key = envServerURL
		value = os.Getenv(key)
	}
	var urls []*url.URL
	for _, field := range strings.Split(value, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		u, err := url.Parse(field)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", key)
		}
		urls = append(urls, u)
	}
	if len(urls) == 0 {
		urls = []*url.URL{defaultServerURL}
	}
	return urls, nil
}

func requestWithContext(ctx context.Context, req *http.Request) *http.Request {
	url := req.URL
	req.URL = nil
	reqCopy := req.WithContext(ctx)
	reqCopy.URL = url
	req.URL = url
	return reqCopy
}

func loadCertificate(path string) (*x509.Certificate, error) {
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for {
		var certBlock *pem.Block
		certBlock, pemBytes = pem.Decode(pemBytes)
		if certBlock == nil {
			return nil, errors.New("missing or invalid certificate")
		}
		if certBlock.Type == "CERTIFICATE" {
			return x509.ParseCertificate(certBlock.Bytes)
		}
	}
}

func verifyPeerCertificate(rawCerts [][]byte, trusted *x509.Certificate) error {
	if len(rawCerts) == 0 {
		return errors.New("missing leaf certificate")
	}
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return errors.Wrap(err, "failed to parse certificate from server")
	}
	if !cert.Equal(trusted) {
		return errors.New("failed to verify server certificate")
	}
	return nil
}

func defaultUserAgent() string {
	return fmt.Sprintf("elasticapm-go/%s go/%s", apmversion.AgentVersion, runtime.Version())
}

func copyHeaders(in http.Header) http.Header {
	out := make(http.Header, len(in))
	for k, vs := range in {
		vsCopy := make([]string, len(vs))
		copy(vsCopy, vs)
		out[k] = vsCopy
	}
	return out
}

type configResponse struct {
	err    error
	attrs  map[string]string
	etag   string
	maxAge time.Duration
}

type cacheControl struct {
	maxAge time.Duration
}

func parseCacheControl(s string) cacheControl {
	fields := strings.SplitN(s, "max-age=", 2)
	if len(fields) < 2 {
		return cacheControl{maxAge: -1}
	}
	s = fields[1]
	if i := strings.IndexRune(s, ','); i != -1 {
		s = s[:i]
	}
	maxAge, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return cacheControl{maxAge: -1}
	}
	return cacheControl{maxAge: time.Duration(maxAge) * time.Second}
}
