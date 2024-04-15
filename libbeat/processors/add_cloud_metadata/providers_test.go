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

package add_cloud_metadata

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func init() {
	os.Unsetenv("BEATS_ADD_CLOUD_METADATA_PROVIDERS")
}

func TestProvidersFilter(t *testing.T) {
	var allLocal []string
	for name, ff := range cloudMetaProviders {
		if ff.Local {
			allLocal = append(allLocal, name)
		}
	}

	cases := map[string]struct {
		config   map[string]interface{}
		env      string
		fail     bool
		expected []string
	}{
		"all with local access only if not configured": {
			config:   map[string]interface{}{},
			expected: allLocal,
		},
		"BEATS_ADD_CLOUD_METADATA_PROVIDERS overrides default": {
			config:   map[string]interface{}{},
			env:      "alibaba, digitalocean",
			expected: []string{"alibaba", "digitalocean"},
		},
		"none if BEATS_ADD_CLOUD_METADATA_PROVIDERS is explicitly set to an empty list": {
			config:   map[string]interface{}{},
			env:      " ",
			expected: nil,
		},
		"fail to load if unknown name is used": {
			config: map[string]interface{}{
				"providers": []string{"unknown"},
			},
			fail: true,
		},
		"only selected": {
			config: map[string]interface{}{
				"providers": []string{"aws", "gcp", "digitalocean"},
			},
		},
		"BEATS_ADD_CLOUD_METADATA_PROVIDERS overrides selected": {
			config: map[string]interface{}{
				"providers": []string{"aws", "gcp", "digitalocean"},
			},
			env:      "alibaba, digitalocean",
			expected: []string{"alibaba", "digitalocean"},
		},
	}

	copyStrings := func(in []string) (out []string) {
		return append(out, in...)
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			rawConfig := conf.MustNewConfigFrom(test.config)
			if test.env != "" {
				t.Setenv("BEATS_ADD_CLOUD_METADATA_PROVIDERS", test.env)
			}

			config := defaultConfig()
			err := rawConfig.Unpack(&config)
			if err == nil && test.fail {
				t.Fatal("Did expect to fail on unpack")
			} else if err != nil && !test.fail {
				t.Fatal("Unpack failed", err)
			} else if test.fail && err != nil {
				return
			}

			// compute list of providers that should have matched
			var expected []string
			if len(test.expected) == 0 && len(config.Providers) > 0 {
				expected = copyStrings(config.Providers)
			} else {
				expected = copyStrings(test.expected)
			}
			sort.Strings(expected)

			var actual []string
			for name := range selectProviders(config.Providers, cloudMetaProviders) {
				actual = append(actual, name)
			}

			sort.Strings(actual)
			assert.Equal(t, expected, actual)
		})
	}
}
