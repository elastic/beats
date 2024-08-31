// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package wireless_device_channel_utilization

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
)

// Uplink contains static device uplink attributes; uplinks are always associated with a device
type WirelessDevicesChannelUtilizationByDevice []struct {
	Serial  meraki.Serial `json:"serial"`
	Mac     string        `json:"mac"`
	Network struct {
		ID string `json:"id"`
	} `json:"network"`
	ByBand []struct {
		Band string `json:"band"`
		Wifi struct {
			Percentage float64 `json:"percentage"`
		} `json:"wifi"`
		NonWifi struct {
			Percentage float64 `json:"percentage"`
		} `json:"nonWifi"`
		Total struct {
			Percentage float64 `json:"percentage"`
		} `json:"total"`
	} `json:"byBand"`
}
