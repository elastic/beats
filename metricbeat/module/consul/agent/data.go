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

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
)

type valueHelper struct {
	Value string
	Unit  string
}

var (
	allowedValues = map[string]valueHelper{
		"consul.autopilot.healthy":         {Value: "autopilot.healthy"},
		"consul.runtime.alloc_bytes":       {Value: "runtime.alloc", Unit: ".bytes"},
		"consul.runtime.total_gc_pause_ns": {Value: "runtime.garbage_collector.pause.total", Unit: ".ns"},
		"consul.runtime.gc_pause_ns":       {Value: "runtime.garbage_collector.pause.current", Unit: ".ns"},
		"consul.runtime.total_gc_runs":     {Value: "runtime.garbage_collector.runs"},
		"consul.runtime.num_goroutines":    {Value: "runtime.goroutines"},
		"consul.runtime.heap_objects":      {Value: "runtime.heap_objects"},
		"consul.runtime.sys_bytes":         {Value: "runtime.sys", Unit: ".bytes"},
		"consul.runtime.malloc_count":      {Value: "runtime.malloc_count"},
	}
	allowedDetailedValues = map[string]valueHelper{
		//"consul.raft.apply":      {Value: "raft.apply"},
		//"consul.raft.commitTime": {Value: "raft.commit_time", Unit: ".ms"},
	}
)

func eventMapping(content []byte) ([]common.MapStr, error) {
	var agent agent

	if err := json.Unmarshal(content, &agent); err != nil {
		return nil, err
	}

	labels := map[string]common.MapStr{}

	for _, gauge := range agent.Gauges {
		metricApply(labels, gauge.consulMetric, gauge.Value)
	}

	for _, point := range agent.Points {
		metricApply(labels, point.consulMetric, point.Value)
	}

	for _, counter := range agent.Counters {
		metricApply(labels, counter.consulMetric, consulDetailedValue(counter))
	}

	for _, sample := range agent.Samples {
		metricApply(labels, sample.consulMetric, consulDetailedValue(sample))
	}

	data := make([]common.MapStr, 0)
	for _, v := range labels {
		data = append(data, v)
	}

	return data, nil
}

func metricApply(labels map[string]common.MapStr, m consulMetric, v interface{}) {
	prettyName := prettyName(m.Name)
	if prettyName == nil {
		//omitting unwanted metric
		return
	}

	labelsCombination := uniqueKeyForLabelMap(m.Labels)

	temp := common.MapStr{}
	if len(m.Labels) != 0 {
		temp.Put("labels", m.Labels)
	}
	key := fmt.Sprintf("%s%s", prettyName.Value, prettyName.Unit)

	var value interface{}
	switch v := v.(type) {
	case consulDetailedValue:
		value = v.Mean
	default:
		value = v
	}

	if _, ok := labels[labelsCombination]; !ok {
		temp.Put(key, value)
		labels[labelsCombination] = temp
	} else {
		labels[labelsCombination].Put(key, value)
	}
}

// prettyName is used to translate a name in Consul metrics to a metric name that follows ES naming conventions
// https://www.elastic.co/guide/en/beats/devguide/current/event-conventions.html
func prettyName(s string) *valueHelper {
	for k, v := range allowedValues {
		if s == k {
			return &v
		}
	}

	for k, v := range allowedDetailedValues {
		if s == k {
			return &v
		}
	}

	return nil
}

// Create a simple unique value for a map of labels without using a hash function
func uniqueKeyForLabelMap(m map[string]string) string {
	mm := common.MapStr{}
	for k, v := range m {
		mm[k] = v
	}

	return mm.String()
}
