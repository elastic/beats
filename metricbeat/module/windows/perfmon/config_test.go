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

// +build windows

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	config := Config{}
	err := config.ValidateConfig()
	assert.Error(t, err, "no perfmon counters or queries have been configured")
	config.Counters = []Counter{
		{
			MeasurementLabel: "processor.time.total.pct",
			Query:            `UDPv4\Datagrams Sent/sec`,
		},
	}
	config.Queries = []Query{
		{
			Name: "UDPv4",
			Counters: []QueryCounter{
				{
					Name: "Datagrams Sent/sec",
				},
			},
		},
	}
	err = config.ValidateConfig()
	assert.NoError(t, err)
	assert.Equal(t, config.Counters[0].Format, "float")
	assert.Equal(t, config.Queries[0].Counters[0].Format, "float")

}
