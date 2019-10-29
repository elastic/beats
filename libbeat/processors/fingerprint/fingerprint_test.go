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

package fingerprint

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestHashMethods(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{
			"md5",
			"3455d980d9c2a5a1c2c0b090a929aa3a",
		},
		{
			"sha1",
			"46de5d8225e75aeedd559c953f100dca41612b18",
		},
		{
			"sha256",
			"4cf8b768ad20266c348d63a6d1ff5d6f6f9ed0f59f5c68ae031b78e3e04c5144",
		},
		{
			"sha384",
			"251b4d77ceea8ad64bf5ed906b5760f9b758af3b30e8f9de5d0d70ec6a2745d25b1be00c5317dc7859256de2d416b179",
		},
		{
			"sha512",
			"903a7f492a22015c89a8e00c40a85da814c2ff42c28cdf1a29495faa8a849eba00449921a75b12c9c212169f100ebf6b05ac8389a8fbfd61cba6026e86a6e2c1",
		},
	}

	for _, test := range tests {
		name := test.method
		if name == "" {
			name = "default"
		}

		name = fmt.Sprintf("testing %v method", name)
		t.Run(name, func(t *testing.T) {
			testEvent := &beat.Event{
				Fields: common.MapStr{
					"field1": "foo",
				},
				Timestamp: time.Now(),
			}

			testConfig, err := common.NewConfigFrom(common.MapStr{
				"fields": []string{"field1"},
				"method": test.method,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, test.expected, v)
		})
	}
}

// TODO: Order of source fields doesn't matter
// TODO: Missing source fields
// TODO: non-scalar fields
// TODO: hashing time fields
// TODO: invalid fingerprinting method in config
// TODO: encoding
