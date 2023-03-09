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

	"github.com/elastic/elastic-agent-libs/config"
)

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
