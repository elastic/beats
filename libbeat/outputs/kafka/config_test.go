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

package kafka

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	lbtesting "github.com/elastic/beats/v7/libbeat/testing"
)

func TestConfigAcceptValid(t *testing.T) {
	tests := map[string]common.MapStr{
		"default config is valid": common.MapStr{},
		"lz4 with 0.11": common.MapStr{
			"compression": "lz4",
			"version":     "0.11",
		},
		"lz4 with 1.0": common.MapStr{
			"compression": "lz4",
			"version":     "1.0.0",
		},
		"Kerberos with keytab": common.MapStr{
			"kerberos": common.MapStr{
				"auth_type":    "keytab",
				"username":     "elastic",
				"keytab":       "/etc/krb5kcd/kafka.keytab",
				"config_path":  "/etc/path/config",
				"service_name": "HTTP/elastic@ELASTIC",
				"realm":        "ELASTIC",
			},
		},
		"Kerberos with user and password pair": common.MapStr{
			"kerberos": common.MapStr{
				"auth_type":    "password",
				"username":     "elastic",
				"password":     "changeme",
				"config_path":  "/etc/path/config",
				"service_name": "HTTP/elastic@ELASTIC",
				"realm":        "ELASTIC",
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c := common.MustNewConfigFrom(test)
			c.SetString("hosts", 0, "localhost")
			cfg, err := readConfig(c)
			if err != nil {
				t.Fatalf("Can not create test configuration: %v", err)
			}
			if _, err := newSaramaConfig(logp.L(), cfg); err != nil {
				t.Fatalf("Failure creating sarama config: %v", err)
			}
		})
	}
}

func TestConfigInvalid(t *testing.T) {
	tests := map[string]common.MapStr{
		"Kerberos with invalid auth_type": common.MapStr{
			"kerberos": common.MapStr{
				"auth_type":    "invalid_auth_type",
				"config_path":  "/etc/path/config",
				"service_name": "HTTP/elastic@ELASTIC",
				"realm":        "ELASTIC",
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c := common.MustNewConfigFrom(test)
			c.SetString("hosts", 0, "localhost")
			_, err := readConfig(c)
			if err == nil {
				t.Fatalf("Can create test configuration from invalid input")
			}
		})
	}
}

func TestBackoffFunc(t *testing.T) {
	lbtesting.SeedPRNG(t)
	err := quick.Check(func(i uint) bool {
		initBackoff := 1 + (i % 5)               // 1 to 5
		maxBackoff := initBackoff + 1 + (i % 45) // initBackoff+1 to 50
		maxRetries := int(1 + (i % 50))          // 1 to 50

		cfg := backoffConfig{
			Init: time.Duration(initBackoff) * time.Second,
			Max:  time.Duration(maxBackoff) * time.Second,
		}

		backoffFn := makeBackoffFunc(cfg)

		for retries := 0; retries < maxRetries; retries++ {
			backoff := backoffFn(retries, maxRetries)

			if !assert.GreaterOrEqual(t, backoff.Milliseconds(), cfg.Init.Milliseconds()/2) {
				t.Logf("init: %v, max: %v, retries: %v", cfg.Init, cfg.Max, retries)
				return false
			}
			if !assert.LessOrEqual(t, backoff.Milliseconds(), cfg.Max.Milliseconds()) {
				t.Logf("init: %v, max: %v, retries: %v", cfg.Init, cfg.Max, retries)
				return false
			}
		}

		return true
	}, &quick.Config{MaxCount: 25})
	require.NoError(t, err)
}
