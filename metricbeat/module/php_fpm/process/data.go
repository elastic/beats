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

package process

import (
	"encoding/json"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/libbeat/common"
)

type phpFpmStatus struct {
	Name      string          `json:"pool"`
	Processes []phpFpmProcess `json:"processes"`
}

type phpFpmProcess struct {
	PID               int     `json:"pid"`
	State             string  `json:"state"`
	StartTime         int     `json:"start time"`
	StartSince        int     `json:"start since"`
	Requests          int     `json:"requests"`
	RequestDuration   int     `json:"request duration"`
	RequestMethod     string  `json:"request method"`
	RequestURI        string  `json:"request uri"`
	ContentLength     int     `json:"content length"`
	User              string  `json:"user"`
	Script            string  `json:"script"`
	LastRequestCPU    float64 `json:"last request cpu"`
	LastRequestMemory int     `json:"last request memory"`
}

func eventsMapping(r mb.ReporterV2, content []byte) {
	var status phpFpmStatus
	err := json.Unmarshal(content, &status)
	if err != nil {
		r.Error(err)
		return
	}
	//remapping process details to match the naming format
	for _, process := range status.Processes {
		proc := common.MapStr{
			"pid":                 process.PID,
			"state":               process.State,
			"start_time":          process.StartTime,
			"start_since":         process.StartSince,
			"requests":            process.Requests,
			"request_duration":    process.RequestDuration,
			"request_method":      process.RequestMethod,
			"request_uri":         process.RequestURI,
			"content_length":      process.ContentLength,
			"user":                process.User,
			"script":              process.Script,
			"last_request_cpu":    process.LastRequestCPU,
			"last_request_memory": process.LastRequestMemory,
		}
		event := mb.Event{MetricSetFields: proc}
		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("pool.name", status.Name)
		r.Event(event)
	}
}
