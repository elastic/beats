// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modifiers

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectInsecure(t *testing.T) {
	cases := []struct {
		Name             string
		Config           map[string]interface{}
		Expected         map[string]interface{}
		VerificationMode tlscommon.TLSVerificationMode
	}{
		{
			Name: "no change on default",
			Config: map[string]interface{}{
				"outputs": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"key": "value",
					},
				},
			},
			Expected: map[string]interface{}{
				"outputs": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"key": "value",
					},
				},
			},
			VerificationMode: tlscommon.VerifyFull, // default
		},
		{
			Name: "inject none",
			Config: map[string]interface{}{
				"outputs": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"key": "value",
					},
				},
			},
			Expected: map[string]interface{}{
				"outputs": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"key":                   "value",
						"ssl.verification_mode": "none",
					},
				},
			},
			VerificationMode: tlscommon.VerifyNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			fn := InjectInsecureOutput(&configuration.FleetAgentConfig{
				Server: &configuration.FleetServerConfig{
					TLS: &tlscommon.Config{
						VerificationMode: tc.VerificationMode,
					},
				},
			})

			ast, err := transpiler.NewAST(tc.Config)
			require.NoError(t, err)
			expectedAST, err := transpiler.NewAST(tc.Expected)
			require.NoError(t, err)

			require.NoError(t, fn(nil, ast))

			visitor := &transpiler.MapVisitor{}
			expectedVisitor := &transpiler.MapVisitor{}

			ast.Accept(visitor)
			expectedAST.Accept(expectedVisitor)

			if !assert.True(t, cmp.Equal(expectedVisitor.Content, visitor.Content)) {
				diff := cmp.Diff(expectedVisitor.Content, visitor.Content)
				if diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

}
