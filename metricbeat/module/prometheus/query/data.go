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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
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

func parseResponse(body []byte, pathConfig QueryConfig) ([]mb.Event, error) {
	var events []mb.Event
	var resultType string
	var convertedMap MapResponse

	// try to convert to array response
	convertedArray, err := convertJSONToArrayResponse(body)
	if err != nil {
		return events, err
	}
	resultType = convertedArray.Data.ResultType

	// check if it is a vector or matrix and unmarshal more
	if resultType == "vector" || resultType == "matrix" {
		convertedMap, err = convertJSONToMapResponse(body)
		if err != nil {
			return events, err
		}
		resultType = convertedMap.Data.ResultType
	}

	switch resultType {
	case "scalar", "string":
		if convertedArray.Data.Results != nil {
			events = append(events, mb.Event{
				Timestamp: getTimestamp(convertedArray.Data.Results[0].(float64)),
				MetricSetFields: common.MapStr{
					"dataType":           resultType,
					pathConfig.QueryName: attemptConvertToNumeric(convertedArray.Data.Results[1].(string)),
				},
			})
		} else {
			return events, errors.New("Could not retrieve results")
		}
	case "vector":
		for _, result := range convertedMap.Data.Results {
			if result.Vector != nil {
				events = append(events, mb.Event{
					Timestamp:    getTimestamp(result.Vector[0].(float64)),
					ModuleFields: common.MapStr{"labels": result.Metric},
					MetricSetFields: common.MapStr{
						"dataType":           resultType,
						pathConfig.QueryName: attemptConvertToNumeric(result.Vector[1].(string)),
					},
				})
			} else {
				return events, errors.New("Could not retrieve results")
			}
		}
	case "matrix":
		for _, result := range convertedMap.Data.Results {
			for _, vector := range result.Vectors {
				if vector != nil {
					events = append(events, mb.Event{
						Timestamp:    getTimestamp(vector[0].(float64)),
						ModuleFields: common.MapStr{"labels": result.Metric},
						MetricSetFields: common.MapStr{
							"dataType":           resultType,
							pathConfig.QueryName: attemptConvertToNumeric(vector[1].(string)),
						},
					})
				} else {
					return events, errors.New("Could not retrieve results")
				}
			}
		}
	default:
		return events, errors.New("Unknown resultType " + resultType)
	}
	return events, nil
}

func convertJSONToMapResponse(body []byte) (MapResponse, error) {
	mapBody := MapResponse{}
	if err := json.Unmarshal(body, &mapBody); err != nil {
		return MapResponse{}, errors.Wrap(err, "Failed to parse api response")
	}
	return mapBody, nil
}

func convertJSONToArrayResponse(body []byte) (ArrayResponse, error) {
	arrayBody := ArrayResponse{}
	if err := json.Unmarshal(body, &arrayBody); err != nil {
		return arrayBody, errors.Wrap(err, "Failed to parse api response")
	}
	if arrayBody.Status == "error" {
		return arrayBody, errors.Errorf("Failed to query")
	}
	return arrayBody, nil
}

func getTimestamp(num float64) time.Time {
	sec := int64(num)
	ns := int64((num - float64(sec)) * 1000)
	return time.Unix(sec, ns)
}

func attemptConvertToNumeric(str string) interface{} {
	if res, err := strconv.Atoi(str); err == nil {
		return res
	} else if res, err := strconv.ParseFloat(str, 64); err == nil {
		return res
	} else {
		return str
	}
}
