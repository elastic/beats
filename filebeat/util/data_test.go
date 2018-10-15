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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
)

func TestNewData(t *testing.T) {
	data := NewData()

	assert.False(t, data.HasEvent())
	assert.False(t, data.HasState())

	data.SetState(file.State{Source: "-"})

	assert.False(t, data.HasEvent())
	assert.True(t, data.HasState())

	data.Event.Fields = common.MapStr{}

	assert.True(t, data.HasEvent())
	assert.True(t, data.HasState())
}

func TestGetEvent(t *testing.T) {
	data := NewData()
	data.Event.Fields = common.MapStr{"hello": "world"}
	out := common.MapStr{"hello": "world"}
	assert.Equal(t, out, data.GetEvent().Fields)
}
