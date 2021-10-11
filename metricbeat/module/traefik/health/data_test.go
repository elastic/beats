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

package health

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {
	// Taken from actual response from a Traefik instance's health API endpoint
	input := map[string]interface{}{
		"pid":               1,
		"uptime":            "16h43m51.452460402s",
		"uptime_sec":        60231.452460402,
		"time":              "2018-06-27 20:59:57.166337808 +0000 UTC m=+60231.514060714",
		"unixtime":          1530133197,
		"status_code_count": map[string]interface{}{},
		"total_status_code_count": map[string]interface{}{
			"200": 17,
			"404": 1,
		},
		"count":                     0,
		"total_count":               18,
		"total_response_time":       "272.119µs",
		"total_response_time_sec":   0.000272119,
		"average_response_time":     "15.117µs",
		"average_response_time_sec": 1.5117e-05,
	}

	event, errors := eventMapping(input)
	assert.Nil(t, errors, "Errors while mapping input to event")

	uptime := event["uptime"].(common.MapStr)
	assert.EqualValues(t, 60231, uptime["sec"])

	response := event["response"].(common.MapStr)
	assert.EqualValues(t, 18, response["count"])

	avgTime := response["avg_time"].(common.MapStr)
	assert.EqualValues(t, 15, avgTime["us"])

	statusCodes := response["status_codes"].(common.MapStr)
	assert.EqualValues(t, 17, statusCodes["200"])
	assert.EqualValues(t, 1, statusCodes["404"])
}
