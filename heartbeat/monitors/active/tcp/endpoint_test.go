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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeEndpoints(t *testing.T) {
	type args struct {
		hosts         []string
		ports         []uint16
		defaultScheme string
	}
	tests := []struct {
		name          string
		args          args
		wantEndpoints []endpoint
		wantErr       bool
	}{
		{
			"hostname",
			args{[]string{"localhost"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tcp", Hostname: "localhost", Ports: []uint16{123}}},
			false,
		},
		{
			"ipv4",
			args{[]string{"1.2.3.4"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tcp", Hostname: "1.2.3.4", Ports: []uint16{123}}},
			false,
		},
		{
			"unbracketed ipv6",
			args{[]string{"::1"}, []uint16{123}, "tcp"},
			[]endpoint{},
			true,
		},
		{
			"bracketed ipv6",
			args{[]string{"[::1]"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tcp", Hostname: "::1", Ports: []uint16{123}}},
			false,
		},
		{
			"url",
			args{[]string{"tls://example.net"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tls", Hostname: "example.net", Ports: []uint16{123}}},
			false,
		},
		{
			"url:port",
			args{[]string{"example.net:999"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tcp", Hostname: "example.net", Ports: []uint16{999}}},
			false,
		},
		{
			"scheme://url:port",
			args{[]string{"tls://example.net:999"}, []uint16{123}, "tcp"},
			[]endpoint{{Scheme: "tls", Hostname: "example.net", Ports: []uint16{999}}},
			false,
		},
		{
			"hybrid",
			args{[]string{
				"localhost",
				"192.168.0.1",
				"[2607:f8b0:4004:814::200e]",
				"example.net:999",
				"tls://example.net:999",
			}, []uint16{123}, "tcp"},
			[]endpoint{
				{Scheme: "tcp", Hostname: "localhost", Ports: []uint16{123}},
				{Scheme: "tcp", Hostname: "192.168.0.1", Ports: []uint16{123}},
				{Scheme: "tcp", Hostname: "2607:f8b0:4004:814::200e", Ports: []uint16{123}},
				{Scheme: "tcp", Hostname: "example.net", Ports: []uint16{999}},
				{Scheme: "tls", Hostname: "example.net", Ports: []uint16{999}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEndpoints, err := makeEndpoints(tt.args.hosts, tt.args.ports, tt.args.defaultScheme)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tt.wantEndpoints, gotEndpoints)
		})
	}
}
