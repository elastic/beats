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

//go:build integration
// +build integration

package eslegclient

import (
	"context"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v8/libbeat/esleg/eslegtest"
	"github.com/elastic/beats/v8/libbeat/outputs"
)

func TestConnect(t *testing.T) {
	conn := getTestingElasticsearch(t)
	err := conn.Connect()
	assert.NoError(t, err)
}

func TestConnectWithProxy(t *testing.T) {
	wrongPort, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go func() {
		c, err := wrongPort.Accept()
		if err == nil {
			// Provoke an early-EOF error on client
			c.Close()
		}
	}()
	defer wrongPort.Close()

	proxy := startTestProxy(t, eslegtest.GetURL())
	defer proxy.Close()

	// Use connectTestEs instead of getTestingElasticsearch to make use of makeES
	client, err := connectTestEs(t, map[string]interface{}{
		"hosts":   "http://" + wrongPort.Addr().String(),
		"timeout": 5, // seconds
	})
	require.NoError(t, err)
	assert.Error(t, client.Connect(), "it should fail without proxy")

	client, err = connectTestEs(t, map[string]interface{}{
		"hosts":     "http://" + wrongPort.Addr().String(),
		"proxy_url": proxy.URL,
		"timeout":   5, // seconds
	})
	require.NoError(t, err)
	assert.NoError(t, client.Connect())
}

func connectTestEs(t *testing.T, cfg interface{}) (*Connection, error) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"username": eslegtest.GetUser(),
		"password": eslegtest.GetPass(),
	})
	require.NoError(t, err)

	tmp, err := common.NewConfigFrom(cfg)
	require.NoError(t, err)

	err = config.Merge(tmp)
	require.NoError(t, err)

	hosts, err := config.String("hosts", -1)
	require.NoError(t, err)

	username, err := config.String("username", -1)
	require.NoError(t, err)

	password, err := config.String("password", -1)
	require.NoError(t, err)

	timeout, err := config.Int("timeout", -1)
	require.NoError(t, err)

	var proxy string
	if config.HasField("proxy_url") {
		proxy, err = config.String("proxy_url", -1)
		require.NoError(t, err)
	}

	s := ConnectionSettings{
		URL:              hosts,
		Username:         username,
		Password:         password,
		CompressionLevel: 3,
	}
	s.Transport.Timeout = time.Duration(timeout) * time.Second

	if proxy != "" {
		proxyURI, err := httpcommon.NewProxyURIFromString(proxy)
		require.NoError(t, err)
		s.Transport.Proxy.URL = proxyURI
	}

	return NewConnection(s)
}

// getTestingElasticsearch creates a test client.
func getTestingElasticsearch(t eslegtest.TestLogger) *Connection {
	conn, err := NewConnection(ConnectionSettings{
		URL:              eslegtest.GetURL(),
		Username:         eslegtest.GetUser(),
		Password:         eslegtest.GetPass(),
		CompressionLevel: 3,
	})
	conn.Transport.Timeout = 60 * time.Second

	eslegtest.InitConnection(t, conn, err)
	return conn
}

func randomClient(grp outputs.Group) outputs.NetworkClient {
	L := len(grp.Clients)
	if L == 0 {
		panic("no elasticsearch client")
	}

	client := grp.Clients[rand.Intn(L)]
	return client.(outputs.NetworkClient)
}

// startTestProxy starts a proxy that redirects all connections to the specified URL
func startTestProxy(t *testing.T, redirectURL string) *httptest.Server {
	t.Helper()

	realURL, err := url.Parse(redirectURL)
	require.NoError(t, err)

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r.Clone(context.Background())
		req.RequestURI = ""
		req.URL.Scheme = realURL.Scheme
		req.URL.Host = realURL.Host

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		for _, header := range []string{"Content-Encoding", "Content-Type"} {
			w.Header().Set(header, resp.Header.Get(header))
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	}))
	return proxy
}
