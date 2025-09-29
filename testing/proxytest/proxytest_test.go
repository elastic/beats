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

package proxytest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

func TestProxy(t *testing.T) {
	proxyCAKey, proxyCACert, _, err := certutil.NewRootCA()
	require.NoError(t, err, "error creating root CA")

	proxyCert, _, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		proxyCAKey,
		proxyCACert)
	require.NoError(t, err, "error creating server certificate")

	proxyCACertPool := x509.NewCertPool()
	proxyCACertPool.AddCert(proxyCACert)

	type setup struct {
		fakeBackendServer      *httptest.Server
		generateTestHttpClient func(t *testing.T, proxy *Proxy) *http.Client
	}
	type testRequest struct {
		method string
		url    string
		body   io.Reader
	}
	type testcase struct {
		name          string
		setup         setup
		proxyOptions  []Option
		proxyStartTLS bool
		request       testRequest
		wantErr       assert.ErrorAssertionFunc
		assertFunc    func(t *testing.T, proxy *Proxy, resp *http.Response)
	}

	testcases := []testcase{
		{
			name: "Basic scenario, no TLS",
			setup: setup{
				fakeBackendServer:      createFakeBackendServer(),
				generateTestHttpClient: nil,
			},
			proxyOptions:  nil,
			proxyStartTLS: false,
			request: testRequest{
				method: http.MethodGet,
				url:    "http://somehost:1234/some/path/here",
				body:   nil,
			},
			wantErr: assert.NoError,
			assertFunc: func(t *testing.T, proxy *Proxy, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				if assert.NotEmpty(t, proxy.ProxiedRequests(), "proxy should have captured at least 1 request") {
					assert.Contains(t, proxy.ProxiedRequests()[0], "/some/path/here")
				}
			},
		},
		{
			name: "TLS scenario, server cert validation",
			setup: setup{
				fakeBackendServer: createFakeBackendServer(),
				generateTestHttpClient: func(t *testing.T, proxy *Proxy) *http.Client {
					proxyURL, err := url.Parse(proxy.URL)
					require.NoErrorf(t, err, "failed to parse proxy URL %q", proxy.URL)

					// Client trusting the proxy cert CA
					return &http.Client{
						Transport: &http.Transport{
							Proxy: http.ProxyURL(proxyURL),
							TLSClientConfig: &tls.Config{
								RootCAs:    proxyCACertPool,
								MinVersion: tls.VersionTLS12,
							},
						},
					}
				},
			},
			proxyOptions: []Option{
				WithServerTLSConfig(&tls.Config{
					ClientCAs:    proxyCACertPool,
					Certificates: []tls.Certificate{*proxyCert},
					MinVersion:   tls.VersionTLS12,
				}),
			},
			proxyStartTLS: true,
			request: testRequest{
				method: http.MethodGet,
				url:    "http://somehost:1234/some/path/here",
				body:   nil,
			},
			wantErr: assert.NoError,
			assertFunc: func(t *testing.T, proxy *Proxy, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				if assert.NotEmpty(t, proxy.ProxiedRequests(), "proxy should have captured at least 1 request") {
					assert.Contains(t, proxy.ProxiedRequests()[0], "/some/path/here")
				}
			},
		},
		{
			name: "mTLS scenario, client and server cert validation",
			setup: setup{
				fakeBackendServer: createFakeBackendServer(),
				generateTestHttpClient: func(t *testing.T, proxy *Proxy) *http.Client {
					proxyURL, err := url.Parse(proxy.URL)
					require.NoErrorf(t, err, "failed to parse proxy URL %q", proxy.URL)

					// Client certificate
					tlsCert, _, err := certutil.GenerateChildCert(
						"localhost",
						[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
						proxyCAKey,
						proxyCACert)
					require.NoError(t, err, "failed generating client certificate")

					// Client with its own certificate and trusting the proxy cert CA
					return &http.Client{
						Transport: &http.Transport{
							Proxy: http.ProxyURL(proxyURL),
							TLSClientConfig: &tls.Config{
								RootCAs: proxyCACertPool,
								Certificates: []tls.Certificate{
									*tlsCert,
								},
								MinVersion: tls.VersionTLS12,
							},
						}}
				},
			},
			proxyOptions: []Option{
				// require client authentication and verify cert
				WithServerTLSConfig(&tls.Config{
					ClientCAs:    proxyCACertPool,
					ClientAuth:   tls.RequireAndVerifyClientCert,
					Certificates: []tls.Certificate{*proxyCert},
					MinVersion:   tls.VersionTLS12,
				}),
			},
			proxyStartTLS: true,
			request: testRequest{
				method: http.MethodGet,
				url:    "http://somehost:1234/some/path/here",
				body:   nil,
			},
			wantErr: assert.NoError,
			assertFunc: func(t *testing.T, proxy *Proxy, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				if assert.NotEmpty(t, proxy.ProxiedRequests(), "proxy should have captured at least 1 request") {
					assert.Contains(t, proxy.ProxiedRequests()[0], "/some/path/here")
				}
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			var proxyOpts []Option

			if tt.setup.fakeBackendServer != nil {
				defer tt.setup.fakeBackendServer.Close()
				serverURL, err := url.Parse(tt.setup.fakeBackendServer.URL)
				require.NoErrorf(t, err, "failed to parse test HTTP server URL %q", tt.setup.fakeBackendServer.URL)
				proxyOpts = append(proxyOpts, WithRewriteFn(func(u *url.URL) {
					// redirect the requests on the proxy itself
					u.Host = serverURL.Host
				}))
			}

			proxyOpts = append(proxyOpts, tt.proxyOptions...)
			proxy := New(t, proxyOpts...)

			if tt.proxyStartTLS {
				t.Log("Starting proxytest with TLS")
				err = proxy.StartTLS()
			} else {
				t.Log("Starting proxytest without TLS")
				err = proxy.Start()
			}
			require.NoError(t, err, "error starting proxytest")

			defer proxy.Close()

			proxyURL, err := url.Parse(proxy.URL)
			require.NoErrorf(t, err, "failed to parse proxy URL %q", proxy.URL)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, tt.request.method, tt.request.url, tt.request.body)
			require.NoError(t, err, "error creating request")

			var client *http.Client
			if tt.setup.generateTestHttpClient != nil {
				client = tt.setup.generateTestHttpClient(t, proxy)
			} else {
				// basic HTTP client using the proxy
				client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
			}

			resp, err := client.Do(req)
			if resp != nil {
				defer resp.Body.Close()
			}
			if tt.wantErr(t, err, "unexpected error return value") && tt.assertFunc != nil {
				tt.assertFunc(t, proxy, resp)
			}
		})
	}

}

func TestHTTPSProxy(t *testing.T) {
	targetHost := "not-a-server.co"
	proxy, client, target := prepareMTLSProxyAndTargetServer(t, targetHost)
	t.Cleanup(func() {
		proxy.Close()
		target.Close()
	})

	tcs := []struct {
		name   string
		target string
		// assertFn should not close the response body
		assertFn func(*testing.T, *http.Response, error)
	}{
		{
			name:   "successful_request",
			target: "https://" + targetHost,
			assertFn: func(t *testing.T, got *http.Response, err error) {
				if !assert.Equal(t, http.StatusOK, got.StatusCode, "unexpected status code") {
					body, err := io.ReadAll(got.Body)
					if err != nil {
						t.Logf("could not read response body")
						t.FailNow()
					}
					_ = got.Body.Close()

					t.Logf("request body: %s", string(body))
				}

			},
		},
		{
			name:   "request_failure",
			target: "https://any.not.target.will.do",
			assertFn: func(t *testing.T, got *http.Response, err error) {
				assert.NoError(t, err, "request to an invalid host should not fail, but succeed with a HTTP error")
				assert.Equal(t, http.StatusBadGateway, got.StatusCode)

				body, err := io.ReadAll(got.Body)
				require.NoError(t, err, "failed reading response body")
				assert.Contains(t, string(body), "failed performing request to target")
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("making request to %q using proxy %q", tc.target, proxy.URL)

			got, err := client.Get(tc.target) //nolint:noctx // it's a test
			require.NoError(t, err, "request should have succeeded")
			defer got.Body.Close()

			// assertFn should not close the response body
			tc.assertFn(t, got, err)
		})
	}
}

func prepareMTLSProxyAndTargetServer(t *testing.T, targetHost string) (*Proxy, http.Client, *httptest.Server) {
	serverCAKey, serverCACert, _, err := certutil.NewRootCA()
	require.NoError(t, err, "error creating root CA")

	serverCert, _, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		serverCAKey,
		serverCACert)
	require.NoError(t, err, "error creating server certificate")
	serverCACertPool := x509.NewCertPool()
	serverCACertPool.AddCert(serverCACert)

	proxyCAKey, proxyCACert, _, err := certutil.NewRootCA()
	require.NoError(t, err, "error creating root CA")

	proxyCert, _, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		proxyCAKey,
		proxyCACert)
	require.NoError(t, err, "error creating server certificate")
	proxyCACertPool := x509.NewCertPool()
	proxyCACertPool.AddCert(proxyCACert)

	clientCAKey, clientCACert, _, err := certutil.NewRootCA()
	require.NoError(t, err, "error creating root CA")
	clientCACertPool := x509.NewCertPool()
	clientCACertPool.AddCert(clientCACert)

	clientCert, _, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		clientCAKey,
		clientCACert)
	require.NoError(t, err, "error creating server certificate")

	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("It works!"))
		}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{*serverCert},
	}
	server.StartTLS()
	t.Logf("target server running on %s", server.URL)

	proxy := New(t,
		WithVerboseLog(),
		WithRequestLog("https", t.Logf),
		WithRewrite(targetHost+":443", server.URL[8:]),
		WithMITMCA(proxyCAKey, proxyCACert),
		WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					RootCAs:    serverCACertPool,
				},
			},
		}),
		WithServerTLSConfig(&tls.Config{
			Certificates: []tls.Certificate{*proxyCert},
			ClientCAs:    clientCACertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}))
	err = proxy.StartTLS()
	require.NoError(t, err, "error starting proxy")
	t.Logf("proxy running on %s", proxy.URL)

	proxyURL, err := url.Parse(proxy.URL)
	require.NoErrorf(t, err, "failed to parse proxy URL %q", proxy.URL)

	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				RootCAs:      proxyCACertPool,
				Certificates: []tls.Certificate{*clientCert},
				MinVersion:   tls.VersionTLS12,
			},
		},
	}

	return proxy, client, server
}

func createFakeBackendServer() *httptest.Server {
	handlerF := func(writer http.ResponseWriter, request *http.Request) {
		// always return HTTP 200
		writer.WriteHeader(http.StatusOK)
	}

	fakeBackendHTTPServer := httptest.NewServer(http.HandlerFunc(handlerF))
	return fakeBackendHTTPServer
}
