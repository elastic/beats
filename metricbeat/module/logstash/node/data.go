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

package node

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/logstash"
)

var (
	schema = s.Schema{
		"id":      c.Str("id"),
		"host":    c.Str("host"),
		"version": c.Str("version"),
		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),
			"pid":     c.Int("pid"),
		}),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	event := mb.Event{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", logstash.ModuleName)

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Logstash Node API response")
	}

	fields, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure applying node schema")
	}

	// Set service ID
	serviceID, err := fields.GetValue("id")
	if err != nil {
		return elastic.MakeErrorForMissingField("id", elastic.Logstash)
	}
	event.RootFields.Put("service.id", serviceID)
	fields.Delete("id")

	// Set service hostname
	host, err := fields.GetValue("host")
	if err != nil {
		return elastic.MakeErrorForMissingField("host", elastic.Logstash)
	}
	event.RootFields.Put("service.hostname", host)
	fields.Delete("host")

	// Set service version
	version, err := fields.GetValue("version")
	if err != nil {
		return elastic.MakeErrorForMissingField("version", elastic.Logstash)
	}
	event.RootFields.Put("service.version", version)
	fields.Delete("version")

	// Set PID
	pid, err := fields.GetValue("jvm.pid")
	if err != nil {
		return elastic.MakeErrorForMissingField("jvm.pid", elastic.Logstash)
	}
	event.RootFields.Put("process.pid", pid)
	fields.Delete("jvm.pid")

	event.MetricSetFields = fields

	r.Event(event)
	return nil
}
