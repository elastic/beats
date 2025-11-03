// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import "time"

// DriverBinaryEntry represents a single entry in the amcache driver binary table.
type DriverBinaryEntry struct {
	Timestamp               time.Time `osquery:"timestamp" format:"unix"`
	DateTime                time.Time `osquery:"date_time" format:"rfc3339" tz:"UTC"`
	KeyName                 string    `osquery:"key_name"`
	DriverName              string    `osquery:"driver_name"`
	Inf                     string    `osquery:"inf"`
	DriverVersion           string    `osquery:"driver_version"`
	Product                 string    `osquery:"product"`
	ProductVersion          string    `osquery:"product_version"`
	WdfVersion              string    `osquery:"wdf_version"`
	DriverCompany           string    `osquery:"driver_company"`
	DriverPackageStrongName string    `osquery:"driver_package_strong_name"`
	Service                 string    `osquery:"service"`
	DriverInBox             string    `osquery:"driver_in_box"`
	DriverSigned            string    `osquery:"driver_signed"`
	DriverIsKernelMode      string    `osquery:"driver_is_kernel_mode"`
	DriverId                string    `osquery:"driver_id"`
	DriverLastWriteTime     string    `osquery:"driver_last_write_time"`
	DriverType              int64     `osquery:"driver_type"`
	DriverTimeStamp         int64     `osquery:"driver_time_stamp"`
	DriverCheckSum          int64     `osquery:"driver_check_sum"`
	ImageSize               int64     `osquery:"image_size"`
}

func (e *DriverBinaryEntry) PostProcess() {
}
