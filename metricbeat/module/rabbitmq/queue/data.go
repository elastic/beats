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

package queue

import (
	"encoding/json"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	schema = s.Schema{
		"name":        c.Str("name"),
		"vhost":       c.Str("vhost"),
		"durable":     c.Bool("durable"),
		"auto_delete": c.Bool("auto_delete"),
		"exclusive":   c.Bool("exclusive"),
		"node":        c.Str("node"),
		"state":       c.Str("state"),
		"arguments": c.Dict("arguments", s.Schema{
			"max_priority": c.Int("x-max-priority", s.Optional),
		}),
		"consumers": s.Object{
			"count": c.Int("consumers"),
			"utilisation": s.Object{
				"pct": c.Int("consumer_utilisation", s.Optional),
			},
		},
		"messages": s.Object{
			"total": s.Object{
				"count": c.Int("messages"),
				"details": c.Dict("messages_details", s.Schema{
					"rate": c.Float("rate"),
				}),
			},
			"ready": s.Object{
				"count": c.Int("messages_ready"),
				"details": c.Dict("messages_ready_details", s.Schema{
					"rate": c.Float("rate"),
				}),
			},
			"unacknowledged": s.Object{
				"count": c.Int("messages_unacknowledged"),
				"details": c.Dict("messages_unacknowledged_details", s.Schema{
					"rate": c.Float("rate"),
				}),
			},
			"persistent": s.Object{
				"count": c.Int("messages_persistent"),
			},
		},
		"memory": s.Object{
			"bytes": c.Int("memory"),
		},
		"disk": s.Object{
			"reads": s.Object{
				"count": c.Int("disk_reads", s.Optional),
			},
			"writes": s.Object{
				"count": c.Int("disk_writes", s.Optional),
			},
		},
	}
)

func eventsMapping(content []byte, r mb.ReporterV2) {
	var queues []map[string]interface{}
	err := json.Unmarshal(content, &queues)
	if err != nil {
		logp.Err("Error: %+v", err)
		r.Error(err)
		return
	}

	var errors multierror.Errors
	for _, queue := range queues {
		err := eventMapping(queue, r)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		r.Error(errors.Err())
	}
}

func eventMapping(queue map[string]interface{}, r mb.ReporterV2) error {
	fields, err := schema.Apply(queue)
	if err != nil {
		return err
	}

	moduleFields := common.MapStr{}
	if v, err := fields.GetValue("vhost"); err == nil {
		moduleFields.Put("vhost", v)
		fields.Delete("vhost")
	}

	if v, err := fields.GetValue("node"); err == nil {
		moduleFields.Put("node.name", v)
		fields.Delete("node")
	}

	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}

	r.Event(event)
	return nil
}
