// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlecloud

import (
	"time"

	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

// MetadataCollectorInputData is a "container" of input data commonly needed for the GCP service's metadata collectors
type MetadataCollectorInputData struct {
	TimeSeries *monitoringpb.TimeSeries
	ProjectID  string
	Zone       string
	Point      *monitoringpb.Point
	Timestamp  *time.Time
}
