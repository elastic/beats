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

//go:build !requirefips

package kafka

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfigAcceptValidKerberos(t *testing.T) {
	tests := map[string]mapstr.M{
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
