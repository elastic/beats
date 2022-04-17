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

package monitoring

import (
	"testing"

	errw "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestOverrideWithCloudSettings(t *testing.T) {
	tests := map[string]struct {
		in               common.MapStr
		out              common.MapStr
		errAssertionFunc assert.ErrorAssertionFunc
	}{
		"cloud_id_no_es_hosts": {
			common.MapStr{
				"cloud.id": "test:bG9jYWxob3N0JGVzY2x1c3RlciRiMGE1N2RhMTkwNzg0MzZmODcwZmQzNTgwZTRhNjE4ZQ==",
			},
			common.MapStr{
				"elasticsearch.hosts": []string{"https://escluster.localhost:443"},
			},
			assert.NoError,
		},
		"cloud_id_with_es_hosts": {
			common.MapStr{
				"cloud.id":            "test:bG9jYWxob3N0JGVzY2x1c3RlciRiMGE1N2RhMTkwNzg0MzZmODcwZmQzNTgwZTRhNjE4ZQ==",
				"elasticsearch.hosts": []string{"foo", "bar"},
			},
			common.MapStr{
				"elasticsearch.hosts": []string{"https://escluster.localhost:443"},
			},
			assert.NoError,
		},
		"cloud_auth_no_es_auth": {
			common.MapStr{
				"cloud.id":   "test:bG9jYWxob3N0JGVzY2x1c3RlciRiMGE1N2RhMTkwNzg0MzZmODcwZmQzNTgwZTRhNjE4ZQ==",
				"cloud.auth": "elastic:changeme",
			},
			common.MapStr{
				"elasticsearch.hosts":    []string{"https://escluster.localhost:443"},
				"elasticsearch.username": "elastic",
				"elasticsearch.password": "changeme",
			},
			assert.NoError,
		},
		"cloud_auth_with_es_auth": {
			common.MapStr{
				"cloud.id":               "test:bG9jYWxob3N0JGVzY2x1c3RlciRiMGE1N2RhMTkwNzg0MzZmODcwZmQzNTgwZTRhNjE4ZQ==",
				"cloud.auth":             "elastic:changeme",
				"elasticsearch.username": "foo",
				"elasticsearch.password": "bar",
			},
			common.MapStr{
				"elasticsearch.hosts":    []string{"https://escluster.localhost:443"},
				"elasticsearch.username": "elastic",
				"elasticsearch.password": "changeme",
			},
			assert.NoError,
		},
		"cloud_auth_no_id": {
			common.MapStr{
				"cloud.auth": "elastic:changeme",
			},
			common.MapStr{
				"cloud.auth": "elastic:changeme",
			},
			func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.EqualError(t, errCloudCfgIncomplete, errw.Cause(err).Error())
			},
		}}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(test.in)
			expected := common.MustNewConfigFrom(test.out)

			err := OverrideWithCloudSettings(cfg)

			test.errAssertionFunc(t, err)
			assert.EqualValues(t, expected, cfg)
		})
	}
}
