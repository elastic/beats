// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_appliance_performance_score

type PerfScore struct {
	PerformanceScore float64 `json:"perfScore"`
}

// DeviceStatus contains dynamic device attributes
type DevicePerformanceScore struct {
	PerformanceScore float64
	HttpStatusCode   int
}
