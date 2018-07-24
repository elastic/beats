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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func TestUpEndpoint(t *testing.T) {
	server := httptest.NewServer(hbtest.HelloWorldHandler)
	defer server.Close()

	port, err := hbtest.ServerPort(server)
	require.NoError(t, err)

	config := common.NewConfig()
	config.SetString("hosts", 0, "localhost")
	config.SetInt("ports", 0, int64(port))

	jobs, err := create(monitors.Info{}, config)
	require.NoError(t, err)

	job := jobs[0]

	event, _, err := job.Run()
	require.NoError(t, err)

	mapvaltest.Test(
		t,
		mapval.Strict(mapval.Compose(
			hbtest.MonitorChecks(
				fmt.Sprintf("tcp-tcp@localhost:%d", port),
				"localhost",
				"127.0.0.1",
				"tcp",
				"up",
			),
			hbtest.TCPChecks(port),
			mapval.Schema(mapval.Map{
				"resolve": mapval.Map{
					"host":   "localhost",
					"ip":     "127.0.0.1",
					"rtt.us": mapval.IsDuration,
				},
			}),
		))(event.Fields),
	)
}
