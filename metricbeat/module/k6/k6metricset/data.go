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

package k6metricset

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Sample struct {
	Value int     `json:"value,omitempty"`
	Count int     `json:"count,omitempty"`
	Rate  float64 `json:"rate,omitempty"`
	Avg   float64 `json:"avg,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Med   float64 `json:"med,omitempty"`
	Min   float64 `json:"min,omitempty"`
	P90   float64 `json:"p(90),omitempty"`
	P95   float64 `json:"p(95),omitempty"`
}

type Metric struct {
	ID         string `json:"id"`
	Attributes struct {
		Sample Sample `json:"sample"`
	} `json:"attributes"`
}

type Data struct {
	Metrics []Metric `json:"data"`
}

func eventMapping(response []byte) (mapstr.M, error) {
	var data Data

	var err error = json.Unmarshal(response, &data)
	if err != nil {
		return nil, fmt.Errorf("JSON unmarshall fail: %w", err)
	}

	event := mapstr.M{
		"data": mapstr.M{
			"metrics": mapstr.M{},
		},
	}

	wantedMetricIDs := []string{"vus", "vus_max", "http_reqs", "http_req_duration", "http_req_connecting", "http_req_receiving",
		"http_req_sending", "http_req_tls_handshaking", "http_req_waiting"}

	contains := func(items []string, item string) bool {
		for _, i := range items {
			if i == item {
				return true
			}
		}
		return false
	}

	for _, metric := range data.Metrics {

		if contains(wantedMetricIDs, metric.ID) {
			sample := metric.Attributes.Sample
			metricFields := mapstr.M{}
			if sample.Rate != 0 {
				metricFields["rate"] = sample.Rate
				metricFields["count"] = sample.Count
			} else {
				if sample.Value != 0 {
					metricFields["value"] = sample.Value
				} else {
					metricFields["avg"] = sample.Avg
					metricFields["max"] = sample.Max
					metricFields["med"] = sample.Med
					metricFields["p(90)"] = sample.P90
					metricFields["p(95)"] = sample.P95

				}

			}
			event["data"].(mapstr.M)["metrics"].(mapstr.M)[metric.ID] = metricFields
		}

	}

	return event, nil
}
