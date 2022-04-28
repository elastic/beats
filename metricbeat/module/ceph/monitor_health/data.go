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

package monitor_health

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Tick struct {
	time.Time
}

var format = "2006-01-02 15:04:05"

func (t *Tick) MarshalJSON() ([]byte, error) {
	return []byte(t.Time.Format(format)), nil
}

func (t *Tick) UnmarshalJSON(b []byte) (err error) {
	b = b[1 : len(b)-1]
	t.Time, err = time.Parse(format, string(b))
	return
}

type StoreStats struct {
	BytesTotal  int64  `json:"bytes_total"`
	BytesLog    int64  `json:"bytes_log"`
	LastUpdated string `json:"last_updated"`
	BytesMisc   int64  `json:"bytes_misc"`
	BytesSSt    int64  `json:"bytes_sst"`
}

type Mon struct {
	LastUpdated  Tick       `json:"last_updated"`
	Name         string     `json:"name"`
	AvailPercent int64      `json:"avail_percent"`
	KbTotal      int64      `json:"kb_total"`
	KbAvail      int64      `json:"kb_avail"`
	Health       string     `json:"health"`
	KbUsed       int64      `json:"kb_used"`
	StoreStats   StoreStats `json:"store_stats"`
}

type HealthServices struct {
	Mons []Mon `json:"mons"`
}

type Health struct {
	HealthServices []HealthServices `json:"health_services"`
}

type Timecheck struct {
	RoundStatus string `json:"round_status"`
	Epoch       int64  `json:"epoch"`
	Round       int64  `json:"round"`
}

type Output struct {
	OverallStatus string    `json:"overall_status"`
	Timechecks    Timecheck `json:"timechecks"`
	Health        Health    `json:"health"`
}

type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) ([]mapstr.M, error) {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get HealthRequest data")
	}

	events := []mapstr.M{}

	for _, HealthService := range d.Output.Health.HealthServices {
		for _, Mon := range HealthService.Mons {
			event := mapstr.M{
				"last_updated": Mon.LastUpdated,
				"name":         Mon.Name,
				"available": mapstr.M{
					"pct": Mon.AvailPercent,
					"kb":  Mon.KbAvail,
				},
				"total": mapstr.M{
					"kb": Mon.KbTotal,
				},
				"health": Mon.Health,
				"used": mapstr.M{
					"kb": Mon.KbUsed,
				},
				"store_stats": mapstr.M{
					"log": mapstr.M{
						"bytes": Mon.StoreStats.BytesLog,
					},
					"misc": mapstr.M{
						"bytes": Mon.StoreStats.BytesMisc,
					},
					"sst": mapstr.M{
						"bytes": Mon.StoreStats.BytesSSt,
					},
					"total": mapstr.M{
						"bytes": Mon.StoreStats.BytesTotal,
					},
					"last_updated": Mon.StoreStats.LastUpdated,
				},
			}

			events = append(events, event)
		}
	}

	return events, nil
}
