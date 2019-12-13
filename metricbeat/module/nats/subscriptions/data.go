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

package subscriptions

import (
	"encoding/json"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	subscriptionsSchema = s.Schema{
		"total": c.Int("num_subscriptions"),
		"cache": s.Object{
			"size":     c.Int("num_cache"),
			"hit_rate": c.Float("cache_hit_rate"),
			"fanout": s.Object{
				"max": c.Int("max_fanout"),
				"avg": c.Float("avg_fanout"),
			},
		},
		"inserts": c.Int("num_inserts"),
		"removes": c.Int("num_removes"),
		"matches": c.Int("num_matches"),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var event mb.Event
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return errors.Wrap(err, "failure parsing Nats subscriptions API response")

	}
	event.MetricSetFields, err = subscriptionsSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying subscriptions schema")
	}

	r.Event(event)
	return nil
}
