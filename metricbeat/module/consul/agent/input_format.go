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

package agent

type consulMetric struct {
	Name   string            `json:"Name"`
	Labels map[string]string `json:"Labels"`
}

type gauge consulSimpleValue

type counter consulDetailedValue

type sample consulDetailedValue

type consulSimpleValue struct {
	consulMetric
	Value float64 `json:"Value"`
}

type consulDetailedValue struct {
	consulMetric
	Count  int     `json:"Count"`
	Rate   float64 `json:"Rate"`
	Sum    float64 `json:"Sum"`
	Min    float64 `json:"Min"`
	Max    float64 `json:"Max"`
	Mean   float64 `json:"Mean"`
	Stddev float64 `json:"Stddev"`
}

type point consulSimpleValue

type agent struct {
	Timestamp string    `json:"Timestamp"`
	Gauges    []gauge   `json:"Gauges"`
	Points    []point   `json:"Points"`
	Counters  []counter `json:"Counters"`
	Samples   []sample  `json:"Samples"`
}
