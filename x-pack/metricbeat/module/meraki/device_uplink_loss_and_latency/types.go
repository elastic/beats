// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_uplink_loss_and_latency

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
)

// Uplink contains static device uplink attributes; uplinks are always associated with a device
type Uplink struct {
	DeviceSerial meraki.Serial
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
