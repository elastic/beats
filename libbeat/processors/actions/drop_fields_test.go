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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestDropFieldRun(t *testing.T) {
	event := &beat.Event{
		Fields: common.MapStr{
			"field": "value",
		},
		Meta: common.MapStr{
			"meta_field": "value",
		},
	}

	t.Run("supports a normal field", func(t *testing.T) {
		p := dropFields{
			Fields: []string{"field"},
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, common.MapStr{}, newEvent.Fields)
		assert.Equal(t, event.Meta, newEvent.Meta)
	})

	t.Run("supports a metadata field", func(t *testing.T) {
		p := dropFields{
			Fields: []string{"@metadata.meta_field"},
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, common.MapStr{}, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
