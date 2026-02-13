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

//go:build !integration

package sniffer

import (
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/protos"
	_ "github.com/elastic/beats/v7/packetbeat/protos/dns"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestDecodersCleanupStopsProtocolJanitors(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	protocols := protos.NewProtocols()
	err := protocols.Init(true, nil, nil, nil, []*conf.C{
		conf.MustNewConfigFrom(map[string]any{
			"type":    "dns",
			"enabled": true,
		}),
	})
	require.NoError(t, err)

	cfg := config.Config{
		Protocols: map[string]*conf.C{
			"icmp": conf.MustNewConfigFrom(map[string]any{
				"enabled": false,
			}),
		},
	}

	makeDecoders := DecodersFor("test", nil, protocols, nil, nil, cfg)
	_, cleanup, err := makeDecoders(layers.LinkTypeEthernet, "lo", 0)
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	cleanup()
}
