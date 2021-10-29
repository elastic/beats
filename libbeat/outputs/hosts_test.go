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

package outputs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestReadHostsList(t *testing.T) {
	tests := map[string]struct {
		cfg       *common.Config
		want      []string
		has_error bool
	}{
		"empty config": {
			cfg:       common.MustNewConfigFrom(common.MapStr{}),
			want:      nil,
			has_error: true,
		},
		"one host": {
			cfg: common.MustNewConfigFrom(common.MapStr{
				"hosts": []string{"10.45.3.2:9220"},
			}),
			want:      []string{"10.45.3.2:9220"},
			has_error: false,
		},
		"two hosts": {
			cfg: common.MustNewConfigFrom(common.MapStr{
				"hosts": []string{"10.45.3.2:9220", "10.45.3.1:9230"},
			}),
			want:      []string{"10.45.3.2:9220", "10.45.3.1:9230"},
			has_error: false,
		},
		"one host 2 worker": {
			cfg: common.MustNewConfigFrom(common.MapStr{
				"hosts":  []string{"10.45.3.2:9220"},
				"worker": 2,
			}),
			want:      []string{"10.45.3.2:9220", "10.45.3.2:9220"},
			has_error: false,
		},
		"two host 2 worker": {
			cfg: common.MustNewConfigFrom(common.MapStr{
				"hosts":  []string{"10.45.3.2:9220", "10.45.3.1:9230"},
				"worker": 2,
			}),
			want:      []string{"10.45.3.2:9220", "10.45.3.2:9220", "10.45.3.1:9230", "10.45.3.1:9230"},
			has_error: false,
		},
		"two host 2 workers": {
			cfg: common.MustNewConfigFrom(common.MapStr{
				"hosts":   []string{"10.45.3.2:9220", "10.45.3.1:9230"},
				"workers": 2,
			}),
			want:      []string{"10.45.3.2:9220", "10.45.3.2:9220", "10.45.3.1:9230", "10.45.3.1:9230"},
			has_error: false,
		},
	}

	for name, tc := range tests {
		got, err := ReadHostList(tc.cfg)
		if tc.has_error {
			assert.NotNil(t, err, name)
		} else {
			assert.Nil(t, err, name)
		}
		assert.EqualValues(t, tc.want, got, name)
	}
}
