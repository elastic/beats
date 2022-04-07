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

package actions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

func TestNetworkDirection(t *testing.T) {
	tests := []struct {
		Source           string
		Destination      string
		InternalNetworks []string
		Direction        string
		Error            bool
	}{
		{"1.1.1.1", "8.8.8.8", []string{"private"}, "external", false},
		{"1.1.1.1", "192.168.1.218", []string{"private"}, "inbound", false},
		{"192.168.1.218", "8.8.8.8", []string{"private"}, "outbound", false},
		{"192.168.1.218", "192.168.1.219", []string{"private"}, "internal", false},
		// early return
		{"1.1.1.1", "8.8.8.8", []string{"foo"}, "", true},
		{"", "192.168.1.219", []string{"private"}, "", false},
		{"foo", "192.168.1.219", []string{"private"}, "", false},
		{"192.168.1.218", "foo", []string{"private"}, "", false},
		{"192.168.1.218", "", []string{"private"}, "", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v -> %v : %v", tt.Source, tt.Destination, tt.Direction), func(t *testing.T) {
			evt := beat.Event{
				Fields: common.MapStr{
					"source":      tt.Source,
					"destination": tt.Destination,
				},
			}
			p, err := NewAddNetworkDirection(common.MustNewConfigFrom(map[string]interface{}{
				"source":            "source",
				"destination":       "destination",
				"target":            "direction",
				"internal_networks": tt.InternalNetworks,
			}))
			require.NoError(t, err)
			observed, err := p.Run(&evt)
			if tt.Error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			enriched, err := observed.Fields.GetValue("direction")
			if tt.Direction == "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.Direction, enriched)
			}
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		evt := beat.Event{
			Meta: common.MapStr{},
			Fields: common.MapStr{
				"source":      "1.1.1.1",
				"destination": "8.8.8.8",
			},
		}
		p, err := NewAddNetworkDirection(common.MustNewConfigFrom(map[string]interface{}{
			"source":            "source",
			"destination":       "destination",
			"target":            "@metadata.direction",
			"internal_networks": "private",
		}))
		require.NoError(t, err)

		expectedMeta := common.MapStr{
			"direction": "external",
		}

		observed, err := p.Run(&evt)
		require.NoError(t, err)
		require.Equal(t, expectedMeta, observed.Meta)
		require.Equal(t, evt.Fields, observed.Fields)
	})
}
