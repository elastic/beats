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

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestMimeTypeFromTo(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "hello world!",
		},
	}
	p, err := NewDetectMimeType(common.MustNewConfigFrom(map[string]interface{}{
		"field":  "foo.bar.baz",
		"target": "bar.baz.zoiks",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	enriched, err := observed.Fields.GetValue("bar.baz.zoiks")
	require.NoError(t, err)
	require.Equal(t, "text/plain; charset=utf-8", enriched)
}

func TestMimeTypeFromToMetadata(t *testing.T) {
	evt := beat.Event{
		Meta: common.MapStr{},
		Fields: common.MapStr{
			"foo.bar.baz": "hello world!",
		},
	}
	expectedMeta := common.MapStr{
		"field": "text/plain; charset=utf-8",
	}
	p, err := NewDetectMimeType(common.MustNewConfigFrom(map[string]interface{}{
		"field":  "foo.bar.baz",
		"target": "@metadata.field",
	}))
	require.NoError(t, err)

	observed, err := p.Run(&evt)
	require.NoError(t, err)
	require.Equal(t, expectedMeta, observed.Meta)
	require.Equal(t, evt.Fields, observed.Fields)
}

func TestMimeTypeTestNoMatch(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": string([]byte{0, 0}),
		},
	}
	p, err := NewDetectMimeType(common.MustNewConfigFrom(map[string]interface{}{
		"field":  "foo.bar.baz",
		"target": "bar.baz.zoiks",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	hasKey, _ := observed.Fields.HasKey("bar.baz.zoiks")
	require.False(t, hasKey)
}
