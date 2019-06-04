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

package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestNewMetricSet(t *testing.T) {
	t.Run("pass in host", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"module":     "redis",
			"metricsets": []string{"info"},
			"hosts": []string{
				"redis://me:secret@localhost:123",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		ms := mbtest.NewReportingMetricSetV2(t, c)
		assert.Equal(t, "secret", ms.HostData().Password)
	})

	t.Run("password in config", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"module":     "redis",
			"metricsets": []string{"info"},
			"hosts": []string{
				"redis://localhost:123",
			},
			"password": "secret",
		})
		if err != nil {
			t.Fatal(err)
		}

		ms := mbtest.NewReportingMetricSetV2(t, c)
		assert.Equal(t, "secret", ms.HostData().Password)
	})
}
