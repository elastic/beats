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

// +build !integration

package jsontransform

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestWriteJSONKeys_NoOverwriteMerges(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"log": common.MapStr{
				"offset": 0,
				"file": common.MapStr{
					"path": "/var/log/log.json",
				},
			},
		},
	}
	keys := common.MapStr{
		"log": common.MapStr{
			"level":  "debug",
			"offset": 10, // not written to event
			"file": common.MapStr{
				"path":   "/var/log/other.json", // not written to event
				"nested": "written",
			},
		},
	}

	WriteJSONKeys(&evt, keys, false, false)

	expected := beat.Event{
		Fields: common.MapStr{
			"log": common.MapStr{
				"level":  "debug",
				"offset": 0,
				"file": common.MapStr{
					"path":   "/var/log/log.json",
					"nested": "written",
				},
			},
		},
	}
	assert.Equal(t, expected, evt)
}
