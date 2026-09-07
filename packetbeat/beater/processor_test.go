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

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestUnknownProtocolsReason(t *testing.T) {
	protos.Register("knowntest", func(testMode bool, results protos.Reporter, watcher *procs.ProcessesWatcher, cfg *conf.C, logger *logp.Logger) (protos.Plugin, error) {
		return nil, nil
	})

	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{
			name: "empty",
			cfg:  config.Config{},
			want: "",
		},
		{
			name: "known_and_icmp",
			cfg: config.Config{
				ProtocolsList: []*conf.C{
					conf.MustNewConfigFrom(map[string]any{"type": "icmp"}),
					conf.MustNewConfigFrom(map[string]any{"type": "knowntest"}),
				},
			},
			want: "",
		},
		{
			name: "unknown_in_list",
			cfg: config.Config{
				ProtocolsList: []*conf.C{
					conf.MustNewConfigFrom(map[string]any{"type": "knowntest"}),
					conf.MustNewConfigFrom(map[string]any{"type": "nosuchproto"}),
				},
			},
			want: "configuration ignored for unknown protocol plugins: [nosuchproto]",
		},
		{
			name: "unknown_in_map",
			cfg: config.Config{
				Protocols: map[string]*conf.C{
					"icmp":  conf.MustNewConfigFrom(map[string]any{"enabled": true}),
					"bogus": conf.MustNewConfigFrom(map[string]any{"enabled": true}),
				},
			},
			want: "configuration ignored for unknown protocol plugins: [bogus]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := unknownProtocolsReason(test.cfg)
			assert.NoError(t, err, "unexpected error extracting protocol types")
			assert.Equal(t, test.want, got, "unexpected degraded reason")
		})
	}
}
