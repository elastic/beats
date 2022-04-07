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

//go:build windows
// +build windows

package perfmon

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/go-ucfg"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	conf := common.MapStr{
		"module":                                 "windows",
		"period":                                 "10s",
		"metricsets":                             []string{"perfmon"},
		"perfmon.group_measurements_by_instance": true,
	}
	c, err := ucfg.NewFrom(conf)
	assert.NoError(t, err)
	var config Config
	err = c.Unpack(&config)
	assert.Error(t, err, "no perfmon counters or queries have been configured")
	conf["perfmon.queries"] = []common.MapStr{
		{
			"object": "Process",
		},
	}
	c, err = ucfg.NewFrom(conf)
	assert.NoError(t, err)
	err = c.Unpack(&config)
	assert.Error(t, err, "missing required field accessing 'perfmon.queries.0.counters'")

	conf["perfmon.queries"] = []common.MapStr{
		{
			"object": "Process",
			"counters": []common.MapStr{
				{
					"name": "Thread Count",
				},
			},
		},
	}
	c, err = ucfg.NewFrom(conf)
	assert.NoError(t, err)
	err = c.Unpack(&config)
	assert.NoError(t, err)
	assert.Equal(t, config.Queries[0].Counters[0].Format, "float")
	assert.Equal(t, config.Queries[0].Namespace, "metrics")
	assert.Equal(t, config.Queries[0].Name, "Process")
	assert.Equal(t, config.Queries[0].Counters[0].Name, "Thread Count")
	assert.True(t, config.GroupMeasurements)

}
