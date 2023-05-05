// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/cloudsql"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/compute"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/redis"
)

// NewMetadataServiceForConfig returns a service to fetch metadata from a config struct. It must return the Compute
// abstraction to fetch metadata, the pubsub abstraction, etc.
func NewMetadataServiceForConfig(c config, serviceName string) (gcp.MetadataService, error) {
	switch serviceName {
	case gcp.ServiceCompute:
		return compute.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.opt...)
	case gcp.ServiceCloudSQL:
		return cloudsql.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.opt...)
	case gcp.ServiceRedis:
		return redis.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.opt...)
	default:
		return nil, nil
	}
}
