// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud/stackdriver/compute"
)

// NewMetadataServiceForConfig returns a service to fetch metadata from a config struct. It must return the Compute
// abstraction to fetch metadata, the pubsub abstraction, etc.
func NewMetadataServiceForConfig(c config, serviceName string) (googlecloud.MetadataService, error) {
	switch serviceName {
	case googlecloud.ServiceCompute:
		return compute.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.opt...)
	default:
		return nil, nil
	}
}
