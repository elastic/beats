// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package syncgateway

type SgResponse struct {
	SyncgatewayChangeCache struct {
		MaxPending float64 `json:"maxPending"`
	} `json:"syncGateway_changeCache"`
	Syncgateway Syncgateway    `json:"syncgateway"`
	MemStats    map[string]any `json:"memstats"`
}

type Syncgateway struct {
	Global struct {
		ResourceUtilization map[string]any `json:"resource_utilization"`
	} `json:"global"`
	PerDb          map[string]map[string]any `json:"per_db"`
	PerReplication map[string]map[string]any `json:"per_replication"`
}
