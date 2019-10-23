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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestMethodDefault(t *testing.T) {
	TestMethodSHA256(t)
}

func TestMethodSHA256(t *testing.T) {
	testConfig, err := common.NewConfigFrom(common.MapStr{
		"fields": []string{"field1"},
		"method": "sha256",
	})
	assert.NoError(t, err)

	p, err := New(testConfig)
	assert.NoError(t, err)

	testEvent := &beat.Event{
		Fields: common.MapStr{
			"field1": "foo",
		},
		Timestamp: time.Now(),
	}

	newEvent, err := p.Run(testEvent)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("fingerprint")
	assert.NoError(t, err)
	assert.Equal(t, "4cf8b768ad20266c348d63a6d1ff5d6f6f9ed0f59f5c68ae031b78e3e04c5144", v)
}
