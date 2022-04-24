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
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func testTCPConfigCheck(t *testing.T, configMap common.MapStr, host string, port uint16) *beat.Event {
	var stats = plugin.NewMultiRegistry(
		[]plugin.StartStopRegistryRecorder{},
		[]plugin.DurationRegistryRecorder{},
	)

	config, err := common.NewConfigFrom(configMap)
	require.NoError(t, err)

	p, err := create("tcp", config)
	require.NoError(t, err)

	sched := schedule.MustParse("@every 1s")
	job := wrappers.WrapCommon(p.Jobs, stdfields.StdMonitorFields{ID: "test", Type: "tcp", Schedule: sched, Timeout: 1}, stats)[0]

	event := &beat.Event{}
	_, err = job(event)
	require.NoError(t, err)

	require.Equal(t, 1, p.Endpoints)

	return event
}

func setupServer(t *testing.T, serverCreator func(http.Handler) (*httptest.Server, error)) (*httptest.Server, uint16, error) {
	server, err := serverCreator(hbtest.HelloWorldHandler(200))
	if err != nil {
		return nil, 0, err
	}

	port, err := hbtest.ServerPort(server)
	if err != nil {
		return nil, 0, err
	}

	return server, port, nil
}

// newHostTestServer starts a server listening on the IP resolved from the host arg
// httptest.NewServer() binds explicitly on 127.0.0.1 (or [::1] if ipv4 is not available).
// The IP resolved from `localhost` can be a different one, like 127.0.1.1.
func newHostTestServer(handler http.Handler, host string) (*httptest.Server, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to listen on host '%s'", host)
	}

	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
	server.Start()

	return server, nil
}
