// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/config"
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
