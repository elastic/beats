// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/cloudsql"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/compute"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/dataproc"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/redis"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataServiceForConfig returns a service to fetch metadata from a config struct. It must return the Compute
// abstraction to fetch metadata, the pubsub abstraction, etc.
func NewMetadataServiceForConfig(c config, serviceName string, cacheRegistry *gcp.CacheRegistry) (gcp.MetadataService, error) {
	if cacheRegistry == nil {
		cacheRegistry = gcp.NewCacheRegistry(logp.NewLogger("gcp-metadata"), time.Hour)
	}

	switch serviceName {
	case gcp.ServiceCompute:
		return compute.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceCloudSQL:
		return cloudsql.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceRedis:
		return redis.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceDataproc:
		return dataproc.NewMetadataService(c.ProjectID, c.Regions, c.organizationID, c.organizationName, c.projectName, c.CollectDataprocUserLabels, cacheRegistry, c.opt...)
	default:
		return nil, nil
	}
}
