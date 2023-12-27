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

package features

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/require"
)

func TestNewConfigFromProto(t *testing.T) {
	source, err := structpb.NewStruct(map[string]interface{}{
		"fqdn": map[string]interface{}{
			"enabled": false,
		},
	})
	require.NoError(t, err)

	tests := map[string]struct {
		protoFeatures *proto.Features
		expected      *config.C
	}{
		"nil": {
			protoFeatures: nil,
			expected:      nil,
		},
		"fqdn_enabled": {
			protoFeatures: &proto.Features{Fqdn: &proto.FQDNFeature{Enabled: true}},
			expected: config.MustNewConfigFrom(`
features:
  fqdn:
    enabled: true
`),
		},
		"fqdn_disabled": {
			protoFeatures: &proto.Features{Fqdn: &proto.FQDNFeature{Enabled: false}},
			expected: config.MustNewConfigFrom(`
features:
  fqdn:
    enabled: false
`),
		},
		"with_source": {
			protoFeatures: &proto.Features{Fqdn: &proto.FQDNFeature{Enabled: true}, Source: source},
			expected: config.MustNewConfigFrom(`
features:
  fqdn:
    enabled: true
`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := NewConfigFromProto(test.protoFeatures)
			require.NoError(t, err)

			require.Equal(t, test.expected, c)
		})
	}

}

func TestFQDN(t *testing.T) {
	tcs := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "FQDN enabled",
			yaml: `
  features:
    fqdn:
      enabled: true`,
			want: true,
		},
		{
			name: "FQDN disabled",
			yaml: `
  features:
    fqdn:
      enabled: false`,
			want: false,
		},
		{
			name: "FQDN only {}",
			yaml: `
  features:
    fqdn: {}`,
			want: true,
		},
		{
			name: "FQDN empty",
			yaml: `
  features:
    fqdn:`,
			want: false,
		},
		{
			name: "FQDN absent",
			yaml: `
  features:`,
			want: false,
		},
		{
			name: "No features",
			yaml: `
  # no features, just a comment`,
			want: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			c, err := config.NewConfigFrom(tc.yaml)
			if err != nil {
				t.Fatalf("could not parse config YAML: %v", err)
			}

			err = UpdateFromConfig(c)
			if err != nil {
				t.Fatalf("UpdateFromConfig failed: %v", err)
			}

			got := FQDN()
			if got != tc.want {
				t.Errorf("want: %t, got %t", tc.want, got)
			}
		})
	}
}

func TestFQDNCallbacks(t *testing.T) {
	cb1Called, cb2Called := false, false

	err := AddFQDNOnChangeCallback(func(new, old bool) {
		cb1Called = true
	}, "cb1")
	require.NoError(t, err)

	err = AddFQDNOnChangeCallback(func(new, old bool) {
		cb2Called = true
	}, "cb2")
	require.NoError(t, err)

	defer func() {
		// Cleanup in case we don't get to the end of
		// this test successfully.
		if _, exists := flags.fqdnCallbacks["cb1"]; exists {
			RemoveFQDNOnChangeCallback("cb1")
		}
		if _, exists := flags.fqdnCallbacks["cb2"]; exists {
			RemoveFQDNOnChangeCallback("cb2")
		}
	}()

	require.Len(t, flags.fqdnCallbacks, 2)
	flags.SetFQDNEnabled(false)
	require.True(t, cb1Called)
	require.True(t, cb2Called)

	RemoveFQDNOnChangeCallback("cb1")
	require.Len(t, flags.fqdnCallbacks, 1)
	RemoveFQDNOnChangeCallback("cb2")
	require.Len(t, flags.fqdnCallbacks, 0)
}

func TestFQDNWHileCallbackBlocked(t *testing.T) {
	blockChan := make(chan struct{})
	willBlockChan := make(chan struct{})
	unblockedChan := make(chan struct{})
	err := AddFQDNOnChangeCallback(func(new, old bool) {
		willBlockChan <- struct{}{}
		t.Logf("callback is currently blocked.")
		<-blockChan
		t.Logf("callback is unblocked.")

	}, "test-TestFQDNWHileCallbackBlocked")
	require.NoError(t, err)

	// Start with FQDN off
	go func() {
		err = UpdateFromConfig(config.MustNewConfigFrom(map[string]interface{}{
			"features.fqdn.enabled": true,
		}))
		unblockedChan <- struct{}{}
	}()

	// callback should be blocking at this point
	t.Logf("Waiting for callback to block...")
	<-willBlockChan
	whileBlocked := FQDN()
	require.True(t, whileBlocked)

	//now unblock
	blockChan <- struct{}{}
	t.Logf("Waiting for callback to unblock...")
	<-unblockedChan
	unblocked := FQDN()
	require.True(t, unblocked)

}
