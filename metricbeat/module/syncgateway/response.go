package syncgateway

type SgResponse struct {
	SyncgatewayChangeCache struct {
		MaxPending float64 `json:"maxPending"`
	} `json:"syncGateway_changeCache"`
	Syncgateway Syncgateway            `json:"syncgateway"`
	MemStats    map[string]interface{} `json:"memstats"`
}

type Syncgateway struct {
	Global struct {
		ResourceUtilization map[string]interface{} `json:"resource_utilization"`
	} `json:"global"`
	PerDb          map[string]map[string]interface{} `json:"per_db"`
	PerReplication map[string]map[string]interface{} `json:"per_replication"`
}
