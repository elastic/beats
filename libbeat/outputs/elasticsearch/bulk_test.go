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

//go:build !integration
// +build !integration

package elasticsearch

import (
	"testing"

	"github.com/menderesk/beats/v7/libbeat/logp"

	"github.com/stretchr/testify/assert"
)

func TestBulkReadToItems(t *testing.T) {
	response := []byte(`{
		"errors": false,
		"items": [
			{"create": {"status": 200}},
			{"create": {"status": 300}},
			{"create": {"status": 400}}
    ]}`)

	reader := newJSONReader(response)

	err := bulkReadToItems(reader)
	assert.NoError(t, err)

	for status := 200; status <= 400; status += 100 {
		err = reader.ExpectDict()
		assert.NoError(t, err)

		kind, raw, err := reader.nextFieldName()
		assert.NoError(t, err)
		assert.Equal(t, mapKeyEntity, kind)
		assert.Equal(t, []byte("create"), raw)

		err = reader.ExpectDict()
		assert.NoError(t, err)

		kind, raw, err = reader.nextFieldName()
		assert.NoError(t, err)
		assert.Equal(t, mapKeyEntity, kind)
		assert.Equal(t, []byte("status"), raw)

		code, err := reader.nextInt()
		assert.NoError(t, err)
		assert.Equal(t, status, code)

		_, _, err = reader.endDict()
		assert.NoError(t, err)

		_, _, err = reader.endDict()
		assert.NoError(t, err)
	}
}

func TestBulkReadItemStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 200}}`)

	reader := newJSONReader(response)
	code, _, err := bulkReadItemStatus(logp.L(), reader)
	assert.NoError(t, err)
	assert.Equal(t, 200, code)
}

func TestESNoErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 200}}`)
	code, msg, err := readStatusItem(response)

	assert.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", msg)
}

func TestES1StyleErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 400, "error": "test error"}}`)
	code, msg, err := readStatusItem(response)

	assert.NoError(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `"test error"`, msg)
}

func TestES2StyleErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 400, "error": {"reason": "test_error"}}}`)
	code, msg, err := readStatusItem(response)

	assert.NoError(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `{"reason": "test_error"}`, msg)
}

func TestES2StyleExtendedErrorStatus(t *testing.T) {
	response := []byte(`
    {
      "create": {
        "status": 400,
        "error": {
          "reason": "test_error",
          "transient": false,
          "extra": null
        }
      }
    }`)
	code, _, err := readStatusItem(response)

	assert.NoError(t, err)
	assert.Equal(t, 400, code)
}

func readStatusItem(in []byte) (int, string, error) {
	reader := newJSONReader(in)
	code, msg, err := bulkReadItemStatus(logp.L(), reader)
	return code, string(msg), err
}
