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

package stats

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/metricbeat/mb"
)

func TestGetNetworkAddress_URL(t *testing.T) {
	hostData := mb.HostData{
		Host: "127.0.0.1:11211",
		URI:  "tcp://127.0.0.1:11211",
	}
	network, address, err := getNetworkAndAddress(hostData)
	require.NoError(t, err)
	require.Equal(t, "tcp", network)
	require.Equal(t, "127.0.0.1:11211", address)
}

func TestGetNetworkAddress_Unix(t *testing.T) {
	hostData := mb.HostData{
		Host: "/tmp/d.sock",
		URI:  "unix:///tmp/d.sock",
	}
	network, address, err := getNetworkAndAddress(hostData)
	require.NoError(t, err)
	require.Equal(t, "unix", network)
	require.Equal(t, "/tmp/d.sock", address)
}
