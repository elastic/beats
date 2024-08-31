// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_status

// DeviceStatus contains dynamic device attributes
type DeviceStatus struct {
	Gateway        string
	IPType         string
	LastReportedAt string
	PrimaryDNS     string
	PublicIP       string
	SecondaryDNS   string
	Status         string // one of ["online", "alerting", "offline", "dormant"]
}
