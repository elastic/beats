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

package monitorstate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

func MakeESLoader(esc *eslegclient.Connection, indexPattern string, beatLocation *config.LocationWithID) StateLoader {
	if indexPattern == "" {
		// Should never happen, but if we ever make a coding error...
		logp.L().Warn("ES state loader initialized with no index pattern, will not load states from ES")
		return NilStateLoader
	}
	return func(sf stdfields.StdMonitorFields) (*State, error) {
		queryMustClauses := []mapstr.M{
			{
				"match": mapstr.M{"monitor.id": sf.ID},
			},
			{
				"match": mapstr.M{"monitor.type": sf.Type},
			},
			{
				"exists": mapstr.M{"field": "state"},
			},
			{
				// Only search the past 6h of data for perf, otherwise we reset the state
				// Monitors should run more frequently than that.
				"range": mapstr.M{"@timestamp": mapstr.M{"gt": "now-6h"}},
			},
		}

		if sf.RunFrom != nil {
			queryMustClauses = append(queryMustClauses, mapstr.M{
				"match": mapstr.M{"observer.name": sf.RunFrom.ID},
			})
		}
		reqBody := mapstr.M{
			"sort": mapstr.M{"@timestamp": "desc"},
			"query": mapstr.M{
				"bool": mapstr.M{
					"must": queryMustClauses,
				},
			},
		}

		status, body, err := esc.Request("POST", strings.Join([]string{"/", indexPattern, "/", "_search", "?size=1"}, ""), "", nil, reqBody)
		if err != nil || status > 299 {
			return nil, fmt.Errorf("error executing state search for %s: %w", sf.ID, err)
		}

		type stateHits struct {
			Hits struct {
				Hits []struct {
					DocId  string `json:"string"`
					Source struct {
						State State `json:"state"`
					} `json:"_source"`
				} `json:"hits"`
			} `json:"hits"`
		}

		sh := stateHits{}
		err = json.Unmarshal(body, &sh)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal state hits for %s: %w", sf.ID, err)
		}

		if len(sh.Hits.Hits) == 0 {
			logp.L().Infof("no previous state found for monitor %s", sf.ID)
			return nil, nil
		}

		state := &sh.Hits.Hits[0].Source.State

		return state, nil
	}
}
