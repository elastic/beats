// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import "time"

// DriverPackageEntry represents a single entry in the amcache inventory driver package table.
// located at Root\\InventoryDriverPackage
type DriverPackageEntry struct {
	Timestamp    time.Time `osquery:"timestamp" format:"unix"`
	DateTime     time.Time `osquery:"date_time" format:"rfc3339" tz:"UTC"`
	KeyName      string    `osquery:"key_name"`
	ClassGuid    string    `osquery:"class_guid"`
	Class        string    `osquery:"class"`
	Directory    string    `osquery:"directory"`
	Date         string    `osquery:"date"`
	Version      string    `osquery:"version"`
	Provider     string    `osquery:"provider"`
	SubmissionId string    `osquery:"submission_id"`
	DriverInBox  string    `osquery:"driver_in_box"`
	Inf          string    `osquery:"inf"`
	FlightIds    string    `osquery:"flight_ids"`
	RecoveryIds  string    `osquery:"recovery_ids"`
	IsActive     string    `osquery:"is_active"`
	Hwids        string    `osquery:"hwids"`
	SYSFILE      string    `osquery:"sysfile"`
}
