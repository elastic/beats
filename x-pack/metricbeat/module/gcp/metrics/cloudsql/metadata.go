// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudsql

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// NewMetadataService returns the specific Metadata service for a GCP CloudSQL resource.
func NewMetadataService(projectID, zone string, region string, regions []string, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID: projectID,
		zone:      zone,
		region:    region,
		regions:   regions,
		opt:       opt,
		instances: make(map[string]*sqladmin.DatabaseInstance),
		logger:    logp.NewLogger("metrics-cloudsql"),
	}, nil
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
	projectID string
	zone      string
	region    string
	regions   []string
	opt       []option.ClientOption
	// NOTE: instances holds data used for all metrics collected in a given period
	// this avoids calling the remote endpoint for each metric, which would take a long time overall
	instances map[string]*sqladmin.DatabaseInstance
	logger    *logp.Logger
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

	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp)

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
	instance, err := s.instance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error trying to get data from instance '%s' %w", instanceID, err)
	}

	cloudsqlMetadata := &cloudsqlMetadata{
		instanceID: instanceID,
		region:     region,
	}

	if instance == nil {
		s.logger.Debugf("couldn't find instance %s, call sqladmin Instances.List", instanceID)
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

func (s *metadataCollector) instance(ctx context.Context, instanceName string) (*sqladmin.DatabaseInstance, error) {
	s.getInstances(ctx)

	instance, ok := s.instances[instanceName]
	if ok {
		return instance, nil
	}

	// Remake the compute instances map to avoid having stale data.
	s.instances = make(map[string]*sqladmin.DatabaseInstance)

	return nil, nil
}

func (s *metadataCollector) getInstances(ctx context.Context) {
	if len(s.instances) > 0 {
		return
	}

	s.logger.Debug("sqladmin Instances.List API")

	service, err := sqladmin.NewService(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error getting client from sqladmin service: %v", err)
		return
	}

	req := service.Instances.List(s.projectID)
	if err := req.Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
		for _, instancesScopedList := range page.Items {
			s.instances[fmt.Sprintf("%s:%s", instancesScopedList.Project, instancesScopedList.Name)] = instancesScopedList
		}
		return nil
	}); err != nil {
		s.logger.Errorf("sqladmin Instances.List error: %v", err)
	}
}
