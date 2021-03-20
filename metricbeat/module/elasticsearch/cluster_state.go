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
package elasticsearch

import (
	"github.com/elastic/beats/v7/metricbeat/helper"
	"sync"
	"time"
)

// FetchPeriodThreshold is the threshold to use when checking if the cluster state call in this cache should be
// fetched again. It's used as a percentage so 0.9 means that cache is invalidated if at least 90% of period time has
// passed. This small threshold is to avoid race conditions where one of the X calls is that happen after each period do
// not fetch new data while other does, becoming unsync. It does not fully solve the problem because the period waits
// aren't guaranteed but it should be mostly enough.
const FetchPeriodThreshold = 0.9

var clusterStateCache = &clusterStatsResponseCache{}

type clusterStatsResponseCache struct {
	sync.Mutex
	lastFetch              time.Time
	configuredModulePeriod time.Duration
	responseData           []byte
}

func GetClusterStateResponseCache(period time.Duration) *clusterStatsResponseCache {
	clusterStateCache.Lock()
	defer clusterStateCache.Unlock()

	if clusterStateCache.configuredModulePeriod == 0 {
		clusterStateCache = &clusterStatsResponseCache{
			Mutex:                  sync.Mutex{},
			lastFetch:              time.Now().Add(-period * 2),
			configuredModulePeriod: period,
			responseData:           nil,
		}
	}

	return clusterStateCache
}

func (c *clusterStatsResponseCache) GetClusterState(http *helper.HTTP, resetURI string, _ []string) ([]byte, error) {
	c.Lock()
	defer c.Unlock()

	elapsedFromLastFetch := time.Since(c.lastFetch)

	//Check the lifetime of current response data
	if elapsedFromLastFetch == 0 || elapsedFromLastFetch >= (time.Duration(float64(c.configuredModulePeriod)*FetchPeriodThreshold)) {
		//Fetch data again
		var err error
		if c.responseData, err = fetchPath(http, resetURI, "_cluster/state", ""); err != nil {
			return nil, err
		}

		c.lastFetch = time.Now()
	}

	return c.responseData, nil
}
