// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var configTests = []struct {
	name    string
	config  map[string]interface{}
	wantErr error
}{
	{
		name: "invalid_oauth2_config",
		config: map[string]interface{}{
			"account_name": "beatsblobnew",
			"auth.oauth2": map[string]interface{}{
				"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
				"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
			},
			"max_workers":   2,
			"poll":          true,
			"poll_interval": "10s",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
		},
		wantErr: fmt.Errorf("client_id, client_secret and tenant_id are required for OAuth2 auth accessing config"),
	},
	{
		name: "valid_oauth2_config",
		config: map[string]interface{}{
			"account_name": "beatsblobnew",
			"auth.oauth2": map[string]interface{}{
				"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
				"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
				"tenant_id":     "87654321-abcd-ef90-1234-fedcba098765",
			},
			"max_workers":   2,
			"poll":          true,
			"poll_interval": "10s",
			"containers": []map[string]interface{}{
				{
					"name": beatsContainer,
				},
			},
		},
	},
}

func TestConfig(t *testing.T) {
	logp.TestingSetup()
	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(test.config)
			conf := config{}
			err := cfg.Unpack(&conf)

			switch {
			case err == nil && test.wantErr != nil:
				t.Fatalf("expected error unpacking config: %v", test.wantErr)
			case err != nil && test.wantErr == nil:
				t.Fatalf("unexpected error unpacking config: %v", err)
			case err != nil && test.wantErr != nil:
				assert.EqualError(t, err, test.wantErr.Error())
			}
		})
	}
}
