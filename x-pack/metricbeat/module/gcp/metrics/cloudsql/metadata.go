// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudsql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1"

	"github.com/elastic/beats/v9/libbeat/common/backoff"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// NewMetadataService returns the specific Metadata service for a GCP CloudSQL resource.
func NewMetadataService(
	ctx context.Context,
	projectID, zone string,
	region string,
	regions []string,
	organizationID, organizationName, projectName string,
	cacheRegistry *gcp.CacheRegistry,
	logger *logp.Logger,
	opt ...option.ClientOption) (gcp.MetadataService, error) {
	mc := &metadataCollector{
		projectID:        projectID,
		projectName:      projectName,
		organizationID:   organizationID,
		organizationName: organizationName,
		zone:             zone,
		region:           region,
		regions:          regions,
		opt:              opt,
		instanceCache:    cacheRegistry.CloudSQL,
		logger:           logger.Named("metrics-cloudsql"),
	}

	// Freshen up the cache, later all we have to do is look up the instance
	err := mc.instanceCache.EnsureFresh(func() (map[string]*sqladmin.DatabaseInstance, error) {
		instances := make(map[string]*sqladmin.DatabaseInstance)
		r := backoff.NewRetryer(3, time.Second, 30*time.Second)

		err := r.Retry(ctx, func() error {
			var err error
			instances, err = mc.fetchCloudSQLInstances(ctx)
			return err
		})

		return instances, err
	})

	return mc, err
}

// cloudsqlMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type cloudsqlMetadata struct {
	region          string
	instanceID      string
	instanceName    string
	databaseVersion string

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID        string
	projectName      string
	organizationID   string
	organizationName string
	zone             string
	region           string
	regions          []string
	opt              []option.ClientOption
	instanceCache    *gcp.Cache[*sqladmin.DatabaseInstance]
	logger           *logp.Logger
}

func getDatabaseNameAndVersion(db string) mapstr.M {
	parts := strings.SplitN(strings.ToLower(db), "_", 2)

	var cloudsqlDb mapstr.M

	switch {
	case db == "SQL_DATABASE_VERSION_UNSPECIFIED":
		cloudsqlDb = mapstr.M{
			"name":    "sql",
			"version": "unspecified",
		}
	case strings.Contains(parts[0], "sqlserver"):
		cloudsqlDb = mapstr.M{
			"name":    strings.ToLower(parts[0]),
			"version": strings.ToLower(parts[1]),
		}
	default:
		version := strings.ReplaceAll(parts[1], "_", ".")
		cloudsqlDb = mapstr.M{
			"name":    strings.ToLower(parts[0]),
			"version": version,
		}
	}

	return cloudsqlDb
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a CloudSQL TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	cloudsqlMetadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceRegion(resp))
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp, s.organizationID, s.organizationName, s.projectName)

	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	cloudsqlMetadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
	cloudsqlMetadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

	if cloudsqlMetadata.databaseVersion != "" {
		err := mapstr.MergeFields(metadataCollectorData.Labels, mapstr.M{
			"cloudsql": getDatabaseNameAndVersion(cloudsqlMetadata.databaseVersion),
		}, true)
		if err != nil {
			s.logger.Warnf("failed merging cloudsql to label fields: %v", err)
		}
	}

	return metadataCollectorData, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["database_id"]
	}

	return ""
}

func (s *metadataCollector) instanceRegion(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels["region"]
	}

	return ""
}

// instanceMetadata returns the labels of an instance
func (s *metadataCollector) instanceMetadata(ctx context.Context, instanceID, region string) (*cloudsqlMetadata, error) {
	cloudsqlMetadata := &cloudsqlMetadata{
		instanceID: instanceID,
		region:     region,
	}

	instance, ok := s.instanceCache.Get(instanceID)
	if !ok {
		s.logger.Warnf("Instance %s not found in cloudsql cache.", instanceID)
		return cloudsqlMetadata, nil
	}

	if instance.DatabaseVersion != "" {
		cloudsqlMetadata.databaseVersion = instance.DatabaseVersion
	}

	if instance.Name != "" {
		cloudsqlMetadata.instanceName = instance.Name
	}

	return cloudsqlMetadata, nil
}

func (s *metadataCollector) fetchCloudSQLInstances(ctx context.Context) (map[string]*sqladmin.DatabaseInstance, error) {
	s.logger.Debug("sqladmin Instances.List API")

	service, err := sqladmin.NewService(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error getting client from sqladmin service: %v", err)
		return nil, err
	}

	fetchedInstances := make(map[string]*sqladmin.DatabaseInstance)

	req := service.Instances.List(s.projectID)
	if err := req.Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, instancesScopedList := range page.Items {
			fetchedInstances[fmt.Sprintf("%s:%s", instancesScopedList.Project, instancesScopedList.Name)] = instancesScopedList
		}
		return nil
	}); err != nil {
		s.logger.Errorf("sqladmin Instances.List error: %v", err)
	}

	return fetchedInstances, nil
}
