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

// +build integration

package esclientleg

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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/esclientleg/eslegtest"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/internal"
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
	_, client := connectTestEs(t, map[string]interface{}{
		"hosts":   "http://" + wrongPort.Addr().String(),
		"timeout": 5, // seconds
	})
	assert.Error(t, client.Connect(), "it should fail without proxy")

	_, client = connectTestEs(t, map[string]interface{}{
		"hosts":     "http://" + wrongPort.Addr().String(),
		"proxy_url": proxy.URL,
		"timeout":   5, // seconds
	})
	assert.NoError(t, client.Connect())
}

func connectTestEs(t *testing.T, cfg interface{}) (outputs.Client, *Client) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"hosts":            eslegtest.GetEsHost(),
		"username":         eslegtest.GetUser(),
		"password":         eslegtest.GetPass(),
		"template.enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	tmp, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = config.Merge(tmp)
	if err != nil {
		t.Fatal(err)
	}

	info := beat.Info{Beat: "libbeat"}
	im, _ := idxmgmt.DefaultSupport(nil, info, nil)
	output, err := makeES(im, info, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatal(err)
	}

	type clientWrap interface {
		outputs.NetworkClient
		Client() outputs.NetworkClient
	}
	client := randomClient(output).(clientWrap).Client().(*Client)

	// Load version number
	client.Connect()

	return client, client
}

// getTestingElasticsearch creates a test client.
func getTestingElasticsearch(t internal.TestLogger) *Client {
	conn, err := NewConnection(ConnectionSettings{
		URL:              internal.GetURL(),
		Username:         internal.GetUser(),
		Password:         internal.GetUser(),
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
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
