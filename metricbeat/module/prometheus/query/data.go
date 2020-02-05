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
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/pkg/errors"
)

// ArrayResponse is for "scalar", "string" type.
type ArrayResponse struct {
	Status string    `json:"status"`
	Data   arrayData `json:"data"`
}
type arrayData struct {
	ResultType string        `json:"resultType"`
	Results    []interface{} `json:"result"`
}

// MapResponse is for "vector", "matrix" type from Prometheus Query API Request
type MapResponse struct {
	Status string  `json:"status"`
	Data   mapData `json:"data"`
}
type mapData struct {
	ResultType string      `json:"resultType"`
	Results    []mapResult `json:"result"`
}
type mapResult struct {
	Metric  map[string]string `json:"metric"`
	Vector  []interface{}     `json:"value"`
	Vectors [][]interface{}   `json:"values"`
}

func (m *MetricSet) parseResponse(body []byte, pathConfig PathConfig) ([]mb.Event, error) {
	var events []mb.Event
	converted, resultType, err := convertJSONToStruct(body)
	if err != nil {
		return events, err
	}
	switch resultType {
	case "scalar", "string":
		res := converted.(ArrayResponse)
		events = append(events, mb.Event{
			Timestamp: getTimestamp(res.Data.Results[0].(float64)),
			MetricSetFields: common.MapStr{
				"dataType":      resultType,
				pathConfig.Name: convertToNumeric(res.Data.Results[1].(string)),
			},
		})
	case "vector":
		res := converted.(MapResponse)
		for _, result := range res.Data.Results {
			events = append(events, mb.Event{
				Timestamp: getTimestamp(result.Vector[0].(float64)),
				MetricSetFields: common.MapStr{
					"labels":        result.Metric,
					"dataType":      resultType,
					pathConfig.Name: convertToNumeric(result.Vector[1].(string)),
				},
			})
		}
	case "matrix":
		res := converted.(MapResponse)
		for _, result := range res.Data.Results {
			for _, vector := range result.Vectors {
				events = append(events, mb.Event{
					Timestamp: getTimestamp(vector[0].(float64)),
					MetricSetFields: common.MapStr{
						"labels":        result.Metric,
						"dataType":      resultType,
						pathConfig.Name: convertToNumeric(vector[1].(string)),
					},
				})
			}
		}
	default:
		return events, errors.New("Unknown resultType " + resultType)
	}
	return events, nil
}

func convertJSONToStruct(body []byte) (interface{}, string, error) {
	arrayBody := ArrayResponse{}
	if err := json.Unmarshal(body, &arrayBody); err != nil {
		return nil, "", errors.Wrap(err, "Failed to parse api response")
	}

	if arrayBody.Data.ResultType == "vector" || arrayBody.Data.ResultType == "matrix" {
		mapBody := MapResponse{}
		if err := json.Unmarshal(body, &mapBody); err != nil {
			return nil, arrayBody.Data.ResultType, errors.Wrap(err, "Failed to parse api response")
		}
		return mapBody, mapBody.Data.ResultType, nil
	}
	return arrayBody, arrayBody.Data.ResultType, nil
}

func getTimestamp(num float64) time.Time {
	sec := int64(num)
	ns := int64((num - float64(sec)) * 1000)
	return time.Unix(sec, ns)
}

func convertToNumeric(str string) interface{} {
	if res, err := strconv.Atoi(str); err == nil {
		return res
	} else if res, err := strconv.ParseFloat(str, 64); err == nil {
		return res
	} else {
		return str
	}
}
