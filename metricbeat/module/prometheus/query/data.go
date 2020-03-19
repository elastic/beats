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
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Response stores the very basic response information to only keep the Status and the ResultType.
type Response struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
	} `json:"data"`
}

// ArrayResponse is for "scalar", "string" type.
// example: {"status":"success","data":{"resultType":"string","result":[1584628642.569,"100"]}}
type ArrayResponse struct {
	Status string    `json:"status"`
	Data   arrayData `json:"data"`
}
type arrayData struct {
	ResultType string        `json:"resultType"`
	Results    []interface{} `json:"result"`
}

// InstantVectorResponse is for "vector" type from Prometheus Query API Request
// Format:
// [
//  {
//    "metric": { "<label_name>": "<label_value>", ... },
//    "value": [ <unix_time>, "<sample_value>" ]
//  },
//  ...
//]
type InstantVectorResponse struct {
	Status string            `json:"status"`
	Data   instantVectorData `json:"data"`
}
type instantVectorData struct {
	ResultType string                `json:"resultType"`
	Results    []instantVectorResult `json:"result"`
}
type instantVectorResult struct {
	Metric map[string]string `json:"metric"`
	Vector []interface{}     `json:"value"`
}

// InstantVectorResponse is for "vector" type from Prometheus Query API Request
// Format:
// [
//  {
//    "metric": { "<label_name>": "<label_value>", ... },
//    "values": [ [ <unix_time>, "<sample_value>" ], ... ]
//  },
//  ...
//]
type RangeVectorResponse struct {
	Status string          `json:"status"`
	Data   rangeVectorData `json:"data"`
}
type rangeVectorData struct {
	ResultType string              `json:"resultType"`
	Results    []rangeVectorResult `json:"result"`
}
type rangeVectorResult struct {
	Metric  map[string]string `json:"metric"`
	Vectors [][]interface{}   `json:"values"`
}

func parseResponse(body []byte, pathConfig QueryConfig) ([]mb.Event, error) {
	var events []mb.Event

	resultType, err := getResultType(body)
	if err != nil {
		return events, err
	}

	switch resultType {
	case "scalar", "string":
		event, err := getEventFromScalarOrString(body, resultType, pathConfig.QueryName)
		if err != nil {
			return events, err
		}
		events = append(events, event)
	case "vector":
		evnts, err := getEventsFromVector(body, resultType, pathConfig.QueryName)
		if err != nil {
			return events, err
		}
		events = append(events, evnts...)
	case "matrix":
		evnts, err := getEventsFromMatrix(body, resultType, pathConfig.QueryName)
		if err != nil {
			return events, err
		}
		events = append(events, evnts...)
	default:
		msg := fmt.Sprintf("Unknown resultType '%v'", resultType)
		return events, errors.New(msg)
	}
	return events, nil
}

func getEventsFromMatrix(body []byte, resultType string, queryName string) ([]mb.Event, error) {
	events := []mb.Event{}
	convertedMap, err := convertJSONToRangeVectorResponse(body)
	if err != nil {
		return events, err
	}
	results := convertedMap.Data.Results
	for _, result := range results {
		for _, vector := range result.Vectors {
			if vector != nil {
				if len(vector) != 2 {
					return []mb.Event{}, errors.New("Could not parse results")
				}
				timestamp, ok := vector[0].(float64)
				if !ok {
					return []mb.Event{}, errors.New("Could not parse timestamp of result")
				}
				events = append(events, mb.Event{
					Timestamp: getTimestamp(timestamp),
					MetricSetFields: common.MapStr{
						"dataType": resultType,
						queryName:  attemptConvertToNumeric(vector[1].(string)),
					},
				})
			} else {
				return []mb.Event{}, errors.New("Could not parse results")
			}
		}
	}
	return events, nil
}

func getEventsFromVector(body []byte, resultType string, queryName string) ([]mb.Event, error) {
	events := []mb.Event{}
	convertedMap, err := convertJSONToInstantVectorResponse(body)
	if err != nil {
		return events, err
	}
	results := convertedMap.Data.Results
	for _, result := range results {
		if result.Vector != nil {
			if len(result.Vector) != 2 {
				return []mb.Event{}, errors.New("Could not parse results")
			}
			timestamp, ok := result.Vector[0].(float64)
			if !ok {
				return []mb.Event{}, errors.New("Could not parse timestamp of result")
			}
			events = append(events, mb.Event{
				Timestamp: getTimestamp(timestamp),
				MetricSetFields: common.MapStr{
					"dataType": resultType,
					queryName:  attemptConvertToNumeric(result.Vector[1].(string)),
				},
			})
		} else {
			return []mb.Event{}, errors.New("Could not parse results")
		}
	}
	return events, nil
}

func getEventFromScalarOrString(body []byte, resultType string, queryName string) (mb.Event, error) {
	convertedArray, err := convertJSONToArrayResponse(body)
	if err != nil {
		return mb.Event{}, err
	}
	if convertedArray.Data.Results != nil {
		if len(convertedArray.Data.Results) != 2 {
			return mb.Event{}, errors.New("Could not parse results")
		}
		timestamp, ok := convertedArray.Data.Results[0].(float64)
		if !ok {
			return mb.Event{}, errors.New("Could not parse timestamp of result")
		}
		return mb.Event{
			Timestamp: getTimestamp(timestamp),
			MetricSetFields: common.MapStr{
				"dataType": resultType,
				queryName:  attemptConvertToNumeric(convertedArray.Data.Results[1].(string)),
			},
		}, nil
	}
	return mb.Event{}, errors.New("Could not parse results")
}

func getResultType(body []byte) (string, error) {
	response := Response{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", errors.Wrap(err, "Failed to parse api response")
	}
	if response.Status == "error" {
		return "", errors.Errorf("Failed to query")
	}
	return response.Data.ResultType, nil
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

func convertJSONToRangeVectorResponse(body []byte) (RangeVectorResponse, error) {
	mapBody := RangeVectorResponse{}
	if err := json.Unmarshal(body, &mapBody); err != nil {
		return RangeVectorResponse{}, errors.Wrap(err, "Failed to parse api response")
	}
	if mapBody.Status == "error" {
		return mapBody, errors.Errorf("Failed to query")
	}
	return mapBody, nil
}

func convertJSONToInstantVectorResponse(body []byte) (InstantVectorResponse, error) {
	mapBody := InstantVectorResponse{}
	if err := json.Unmarshal(body, &mapBody); err != nil {
		return InstantVectorResponse{}, errors.Wrap(err, "Failed to parse api response")
	}
	if mapBody.Status == "error" {
		return mapBody, errors.Errorf("Failed to query")
	}
	return mapBody, nil
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
