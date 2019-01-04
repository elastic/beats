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

package subsz

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
)

type Subsz struct {
	NumSubscriptionsIn int `json:"num_subscriptions,omitempty"`
	NumSubscriptions   int `json:"total"`
	NumCacheIn         int `json:"num_cache,omitempty"`
	NumCache           int `json:"cache.size"`
	NumInsertsIn       int `json:"num_inserts,omitempty"`
	NumInserts         int `json:"inserts"`
	NumRemovesIn       int `json:"num_removes,omitempty"`
	NumRemoves         int `json:"removes"`
	NumMatchesIn       int `json:"num_matches,omitempty"`
	NumMatches         int `json:"matches"`
	CacheHitRateIn     int `json:"cache_hit_rate,omitempty"`
	CacheHitRate       int `json:"cache.hit_rate"`
	MaxFanoutIn        int `json:"max_fanout,omitempty"`
	MaxFanout          int `json:"cache.fanout.max"`
	AvgFanoutIn        int `json:"avg_fanout,omitempty"`
	AvgFanout          int `json:"cache.fanout.avg"`
}

func eventMapping(content []byte) common.MapStr {
	var data Subsz
	json.Unmarshal(content, &data)

	data.NumSubscriptions = data.NumSubscriptionsIn
	data.NumSubscriptionsIn = 0
	data.NumCache = data.NumCacheIn
	data.NumCacheIn = 0
	data.NumInserts = data.NumInsertsIn
	data.NumInsertsIn = 0
	data.NumRemoves = data.NumRemovesIn
	data.NumRemovesIn = 0
	data.NumMatches = data.NumMatchesIn
	data.NumMatchesIn = 0
	data.CacheHitRate = data.CacheHitRateIn
	data.CacheHitRateIn = 0
	data.MaxFanout = data.MaxFanoutIn
	data.MaxFanoutIn = 0
	data.AvgFanout = data.AvgFanoutIn
	data.AvgFanoutIn = 0

	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
