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
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/internal/testutil"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfigAcceptValid(t *testing.T) {
	tests := map[string]mapstr.M{
		"lz4 with 0.11": mapstr.M{
			"compression": "lz4",
			"version":     "0.11",
			"topic":       "foo",
		},
		"lz4 with 1.0": mapstr.M{
			"compression": "lz4",
			"version":     "1.0.0",
			"topic":       "foo",
		},
		"Kerberos with keytab": mapstr.M{
			"topic": "foo",
			"kerberos": mapstr.M{
				"auth_type":    "keytab",
				"username":     "elastic",
				"keytab":       "/etc/krb5kcd/kafka.keytab",
				"config_path":  "/etc/path/config",
				"service_name": "HTTP/elastic@ELASTIC",
				"realm":        "ELASTIC",
			},
		},
		"Kerberos with user and password pair": mapstr.M{
			"topic": "foo",
			"kerberos": mapstr.M{
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
			c := config.MustNewConfigFrom(test)
			if err := c.SetString("hosts", 0, "localhost"); err != nil {
				t.Fatalf("could not set 'hosts' on config: %s", err)
			}
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
	tests := map[string]mapstr.M{
		"Kerberos with invalid auth_type": mapstr.M{
			"kerberos": mapstr.M{
				"auth_type":    "invalid_auth_type",
				"config_path":  "/etc/path/config",
				"service_name": "HTTP/elastic@ELASTIC",
				"realm":        "ELASTIC",
			},
		},
		// The default config does not set `topic` nor `topics`.
		"No topics or topic provided": mapstr.M{},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c := config.MustNewConfigFrom(test)
			if err := c.SetString("hosts", 0, "localhost"); err != nil {
				t.Fatalf("could not set 'hosts' on config: %s", err)
			}
			_, err := readConfig(c)
			if err == nil {
				t.Fatalf("Can create test configuration from invalid input")
			}
		})
	}
}

func TestInvalidConfigUnderElasticAgent(t *testing.T) {
	oldUnderAgent := management.UnderAgent()
	t.Cleanup(func() {
		// Restore the previous value
		management.SetUnderAgent(oldUnderAgent)
	})

	management.SetUnderAgent(true)
	tests := map[string]mapstr.M{
		"topics is provided": mapstr.M{
			"topics": []string{"foo", "bar"},
		},
		// The default config does not set `topic` not `topics`.
		"No topics or topic provided": mapstr.M{},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			c := config.MustNewConfigFrom(test)
			if err := c.SetString("hosts", 0, "localhost"); err != nil {
				t.Fatalf("could not set 'hosts' on config: %s", err)
			}
			_, err := readConfig(c)
			if err == nil {
				t.Fatalf("Can create test configuration from invalid input")
			}
		})
	}

}

func TestBackoffFunc(t *testing.T) {
	testutil.SeedPRNG(t)
	tests := map[int]backoffConfig{
		15: {Init: 1 * time.Second, Max: 60 * time.Second},
		7:  {Init: 2 * time.Second, Max: 20 * time.Second},
		4:  {Init: 5 * time.Second, Max: 7 * time.Second},
	}

	for numRetries, backoffCfg := range tests {
		t.Run(fmt.Sprintf("%v_retries", numRetries), func(t *testing.T) {
			backoffFn := makeBackoffFunc(backoffCfg)

			prevBackoff := backoffCfg.Init
			for retries := 1; retries <= numRetries; retries++ {
				backoff := prevBackoff * 2

				expectedBackoff := math.Min(float64(backoff), float64(backoffCfg.Max))
				actualBackoff := backoffFn(retries, 50)

				if !((expectedBackoff/2 <= float64(actualBackoff)) && (float64(actualBackoff) <= expectedBackoff)) {
					t.Fatalf("backoff '%v' not in expected range [%v, %v] (retries: %v)", actualBackoff, expectedBackoff/2, expectedBackoff, retries)
				}

				prevBackoff = backoff
			}

		})
	}
}

func TestTopicSelection(t *testing.T) {
	cases := map[string]struct {
		cfg   map[string]interface{}
		event beat.Event
		want  string
	}{
		"topic configured": {
			cfg:  map[string]interface{}{"topic": "test"},
			want: "test",
		},
		"topic must keep case": {
			cfg:  map[string]interface{}{"topic": "Test"},
			want: "Test",
		},
		"topics setting": {
			cfg: map[string]interface{}{
				"topics": []map[string]interface{}{{"topic": "test"}},
			},
			want: "test",
		},
		"topics setting must keep case": {
			cfg: map[string]interface{}{
				"topics": []map[string]interface{}{{"topic": "Test"}},
			},
			want: "Test",
		},
		"use event field": {
			cfg: map[string]interface{}{"topic": "test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "from-event"},
			},
			want: "test-from-event",
		},
		"use event field must keep case": {
			cfg: map[string]interface{}{"topic": "Test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "From-Event"},
			},
			want: "Test-From-Event",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			test := test
			selector, err := buildTopicSelector(config.MustNewConfigFrom(test.cfg))
			if err != nil {
				t.Fatalf("Failed to parse configuration: %v", err)
			}

			got, err := selector.Select(&test.event)
			if err != nil {
				t.Fatalf("Failed to create topic name: %v", err)
			}

			if test.want != got {
				t.Errorf("Pipeline name missmatch (want: %v, got: %v)", test.want, got)
			}
		})
	}
}
