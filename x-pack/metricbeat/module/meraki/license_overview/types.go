// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package license_overview

// device unique identifier
type Serial string

// CoterminationLicense have a common expiration date and are reported per device model
type CoterminationLicense struct {
	ExpirationDate string
	DeviceModel    string
	Status         string
	Count          interface{}
}

// PerDeviceLicense are reported by license state with details on expiration and activations
type PerDeviceLicense struct {
	State                   string // one of ["Active", "Expired", "RecentlyQueued", "Expiring", "Unused", "UnusedActive", "Unassigned"]
	Count                   *int
	ExpirationState         string // one of ["critial", "warning"] (only for Expiring licenses)
	ExpirationThresholdDays *int   // only for Expiring licenses
	SoonestActivationDate   string // only for Unused licenses
	SoonestActivationCount  *int   // only for Unused licenses
	OldestActivationDate    string // only for UnusedActive licenses
	OldestActivationCount   *int   // only for UnusedActive licenses
	Type                    string // one of ["ENT", "UPGR", "ADV"] (only for Unassigned licenses)
}

// SystemManagerLicense reports counts for seats and devices
type SystemsManagerLicense struct {
	ActiveSeats            *int
	OrgwideEnrolledDevices *int
	TotalSeats             *int
	UnassignedSeats        *int
}
