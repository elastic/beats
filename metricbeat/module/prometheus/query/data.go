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

package query

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// PromResponseBody for Prometheus Query API Request
type PromResponseBody struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}
type PrometheusData struct {
	ResultType string   `json:"resultType"`
	Results    []Result `json:"result"`
}
type Result struct {
	Metric  interface{}   `json:"metric"`
	Vectors []interface{} `json:"value"`
}

func (m *MetricSet) parseResponse(body []byte, pathConfig PathConfig) mb.Event {
	var event common.MapStr
	var res PromResponseBody
	if err := json.Unmarshal(body, &res); err != nil {
		m.Logger().Error("Failed to parsing api response ", err)
	}

	// Check if there is vector array.
	// Vector [ <unix_timestamp>, "<query_result>" ] is not acceptable for Elasticsearch.
	// Because there are two types in one array.
	// So change Vector to [ "<unix_timestamp", "query_result" ]
	if res.Data.ResultType == "vector" {
		for _, res := range res.Data.Results {
			if len(res.Vectors) != 0 {
				res.Vectors[0] = fmt.Sprintf("%f", res.Vectors[0])
			}
		}
	}

	event = common.MapStr{
		pathConfig.Name: res,
	}

	return mb.Event{
		MetricSetFields: event,
		Namespace:       "prometheus.query",
	}
}
