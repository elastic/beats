// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/cloudsql"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/compute"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp/metrics/redis"
)

// NewMetadataServiceForConfig returns a service to fetch metadata from a config struct. It must return the Compute
// abstraction to fetch metadata, the pubsub abstraction, etc.
func NewMetadataServiceForConfig(ctx context.Context, c config, serviceName string, cacheRegistry *gcp.CacheRegistry) (gcp.MetadataService, error) {
	switch serviceName {
	case gcp.ServiceCompute:
		return compute.NewMetadataService(ctx, c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceCloudSQL:
		return cloudsql.NewMetadataService(ctx, c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceRedis:
<<<<<<< HEAD
		return redis.NewMetadataService(c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, c.opt...)
=======
		return redis.NewMetadataService(ctx, c.ProjectID, c.Zone, c.Region, c.Regions, c.organizationID, c.organizationName, c.projectName, cacheRegistry, c.opt...)
	case gcp.ServiceDataproc:
		return dataproc.NewMetadataService(ctx, c.ProjectID, c.Regions, c.organizationID, c.organizationName, c.projectName, c.CollectDataprocUserLabels, cacheRegistry, c.opt...)
>>>>>>> 6b6941eed ([gcp] Add metadata cache (#44432))
	default:
		return nil, nil
	}
}
