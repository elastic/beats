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

package beat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFQDNAwareHostname(t *testing.T) {
	info := Info{
		Hostname: "foo",
		FQDN:     "foo.bar.internal",
	}
	cases := map[string]struct {
		useFQDN  bool
		envValue string
		want     string
	}{
		"fqdn_flag_enabled": {
			useFQDN: true,
			want:    "foo.bar.internal",
		},
		"fqdn_flag_disabled": {
			useFQDN: false,
			want:    "foo",
		},
		"env_override_takes_precedence_over_fqdn": {
			useFQDN:  true,
			envValue: "my-node",
			want:     "my-node",
		},
		"env_override_takes_precedence_over_hostname": {
			useFQDN:  false,
			envValue: "my-node",
			want:     "my-node",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				t.Setenv(EnvHostName, tc.envValue)
			}
			got := info.FQDNAwareHostname(tc.useFQDN)
			require.Equal(t, tc.want, got)
		})
	}
}
