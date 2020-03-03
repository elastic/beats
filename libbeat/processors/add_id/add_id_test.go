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

package add_id

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTargetField(t *testing.T) {
	p, err := New(common.MustNewConfigFrom(nil))
	assert.NoError(t, err)

	testEvent := &beat.Event{}

	newEvent, err := p.Run(testEvent)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("@metadata._id")
	assert.NoError(t, err)
	assert.NotEmpty(t, v)
}

func TestNonDefaultTargetField(t *testing.T) {
	cfg := common.MustNewConfigFrom(common.MapStr{
		"target_field": "foo",
	})
	p, err := New(cfg)
	assert.NoError(t, err)

	testEvent := &beat.Event{
		Fields: common.MapStr{},
	}

	newEvent, err := p.Run(testEvent)
	assert.NoError(t, err)

	v, err := newEvent.GetValue("foo")
	assert.NoError(t, err)
	assert.NotEmpty(t, v)

	v, err = newEvent.GetValue("@metadata._id")
	assert.NoError(t, err)
	assert.Empty(t, v)
}
