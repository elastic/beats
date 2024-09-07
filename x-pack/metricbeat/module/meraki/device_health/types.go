// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"time"
)

// device unique identifier
type Serial string

// Device contains static device attributes (i.e. dimensions)
type Device struct {
	Address     string
	Details     map[string]string
	Firmware    string
	Imei        *float64
	LanIP       string
	Location    []*float64
	Mac         string
	Model       string
	Name        string
	NetworkID   string
	Notes       string
	ProductType string // one of ["appliance", "camera", "cellularGateway", "secureConnect", "sensor", "switch", "systemsManager", "wireless", "wirelessController"]
	Serial      string
	Tags        []string
}

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

// Uplink contains static device uplink attributes; uplinks are always associated with a device
type Uplink struct {
	DeviceSerial Serial
	IP           string
	Interface    string
	Metrics      []*UplinkMetric
}

// UplinkMetric contains timestamped device uplink metric data points
type UplinkMetric struct {
	Timestamp   time.Time
	LossPercent *float64
	LatencyMs   *float64
}

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

type PerfScore struct {
	PerformanceScore float64 `json:"perfScore"`
}

// DeviceStatus contains dynamic device attributes
type DevicePerformanceScore struct {
	PerformanceScore float64
	HttpStatusCode   int
}

// Uplink contains static device uplink attributes; uplinks are always associated with a device
type WirelessDevicesChannelUtilizationByDevice []struct {
	Serial  Serial `json:"serial"`
	Mac     string `json:"mac"`
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
