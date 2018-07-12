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

package pending_tasks

import (
	"encoding/json"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"insert_order":     c.Int("insert_order"),
		"priority":         c.Str("priority"),
		"source":           c.Str("source"),
		"time_in_queue.ms": c.Int("time_in_queue_millis"),
	}
)

func eventsMapping(content []byte, applyOpts ...s.ApplyOption) ([]common.MapStr, error) {
	tasksStruct := struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}{}

	if err := json.Unmarshal(content, &tasksStruct); err != nil {
		return nil, err
	}
	if tasksStruct.Tasks == nil {
		return nil, s.NewKeyNotFoundError("tasks")
	}

	var events []common.MapStr
	var errors multierror.Errors

	opts := append(applyOpts, s.AllRequired)
	for _, task := range tasksStruct.Tasks {
		event, err := schema.Apply(task, opts...)
		if err != nil {
			errors = append(errors, err)
		}
		events = append(events, event)
	}

	return events, errors.Err()
}
