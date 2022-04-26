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

package tcp

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/armon/go-socks5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

func TestSocks5Job(t *testing.T) {
	scenarios := []struct {
		name          string
		localResolver bool
	}{
		{
			name:          "using local resolver",
			localResolver: true,
		},
		{
			name:          "not using local resolver",
			localResolver: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			host, port, ip, closeEcho, err := startEchoServer(t)
			require.NoError(t, err)
			//nolint:errcheck // There are no new changes to this line but
			// linter has been activated in the meantime. We'll cleanup separately.
			defer closeEcho()

			_, proxyPort, proxyIp, closeProxy, err := startSocks5Server(t)
			require.NoError(t, err)
			//nolint:errcheck // There are no new changes to this line but
			// linter has been activated in the meantime. We'll cleanup separately.
			defer closeProxy()

			proxyURL := &url.URL{Scheme: "socks5", Host: net.JoinHostPort(proxyIp, fmt.Sprint(proxyPort))}
			configMap := common.MapStr{
				"hosts":                    host,
				"ports":                    port,
				"timeout":                  "1s",
				"proxy_url":                proxyURL.String(),
				"proxy_use_local_resolver": scenario.localResolver,
				"check.receive":            "echo123",
				"check.send":               "echo123",
			}
			event := testTCPConfigCheck(t, configMap, host, port)

			testslike.Test(
				t,
				lookslike.Strict(lookslike.Compose(
					hbtest.BaseChecks(ip, "up", "tcp"),
					hbtest.RespondingTCPChecks(),
					hbtest.SimpleURLChecks(t, "tcp", host, port),
					hbtest.SummaryChecks(1, 0),
					hbtest.ResolveChecks(ip),
					lookslike.MustCompile(map[string]interface{}{
						"tcp": map[string]interface{}{
							"rtt.validate.us": hbtestllext.IsInt64,
						},
						"socks5": map[string]interface{}{
							"rtt.connect.us": hbtestllext.IsInt64,
						},
					}),
				)),
				event.Fields,
			)
		})
	}
}

func startSocks5Server(t *testing.T) (host string, port uint16, ip string, close func() error, err error) {
	//nolint:goconst // Test variable.
	host = "localhost"
	config := &socks5.Config{}
	server, err := socks5.New(config)
	if err != nil {
		return "", 0, "", nil, err
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", 0, "", nil, err
	}
	ip, portStr, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	portUint64, err := strconv.ParseUint(portStr, 10, 16)
	require.NoError(t, err)
	if err != nil {
		listener.Close()
		return "", 0, "", nil, err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := server.Serve(listener); err != nil {
			debugf("Error in SOCKS5 Test Server %v", err)
		}
		wg.Done()
	}()

	return host, uint16(portUint64), ip, func() error {
		err := listener.Close()
		if err != nil {
			return err
		}
		wg.Wait()
		return nil
	}, nil
}
