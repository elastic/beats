// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/x-pack/metricbeat/module/googlecloud"
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(projectID, zone string, opt ...option.ClientOption) (googlecloud.MetadataService, error) {
	return &metadataCollector{
		projectID:     projectID,
		zone:          zone,
		opt:           opt,
		instanceCache: common.NewCache(30*time.Second, 13),
	}, nil
}

// computeMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type computeMetadata struct {
	projectID   string
	zone        string
	instanceID  string
	machineType string

	ts *monitoringpb.TimeSeries

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID string
	zone      string
	opt       []option.ClientOption

	computeMetadata *computeMetadata

	instanceCache *common.Cache
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Compute TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (googlecloud.MetadataCollectorData, error) {
	if s.computeMetadata == nil {
		_, err := s.instanceMetadata(ctx, s.instanceID(resp), s.zone)
		if err != nil {
			return googlecloud.MetadataCollectorData{}, err
		}
	}

	stackdriverLabels := googlecloud.NewStackdriverMetadataServiceForTimeSeries(resp)
	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return googlecloud.MetadataCollectorData{}, err
	}

	if resp.Resource != nil && resp.Resource.Labels != nil {
		metadataCollectorData.ECS.Put(googlecloud.ECSCloudInstanceIDKey, resp.Resource.Labels[googlecloud.TimeSeriesResponsePathForECSInstanceID])
	}

	if resp.Metric.Labels != nil {
		metadataCollectorData.ECS.Put(googlecloud.ECSCloudInstanceNameKey, resp.Metric.Labels[googlecloud.TimeSeriesResponsePathForECSInstanceName])
	}

	if s.computeMetadata.machineType != "" {
		lastIndex := strings.LastIndex(s.computeMetadata.machineType, "/")
		metadataCollectorData.ECS.Put(googlecloud.ECSCloudMachineTypeKey, s.computeMetadata.machineType[lastIndex+1:])
	}

	s.computeMetadata.Metrics = metadataCollectorData.Labels[googlecloud.LabelMetrics]
	s.computeMetadata.System = metadataCollectorData.Labels[googlecloud.LabelSystem]

	if s.computeMetadata.User != nil {
		metadataCollectorData.Labels[googlecloud.LabelUser] = s.computeMetadata.User
	}

	/*
		Do not collect meta for now, as it can contain sensitive info
		TODO revisit this and make meta available through whitelisting
		if s.computeMetadata.Metadata != nil {
			metadataCollectorData.Labels[googlecloud.LabelMetadata] = s.computeMetadata.Metadata
		}
	*/

	return metadataCollectorData, nil
}

// instanceMetadata returns the labels of an instance
func (s *metadataCollector) instanceMetadata(ctx context.Context, instanceID, zone string) (*computeMetadata, error) {
	i, err := s.instance(ctx, instanceID, zone)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to get data from instance '%s' in zone '%s'", instanceID, zone)
	}

	s.computeMetadata = &computeMetadata{
		instanceID: instanceID,
		zone:       zone,
	}

	if i.Labels != nil {
		s.computeMetadata.User = i.Labels
	}

	if i.MachineType != "" {
		s.computeMetadata.machineType = i.MachineType
	}

	if i.Metadata != nil && i.Metadata.Items != nil {
		s.computeMetadata.Metadata = make(map[string]string)

		for _, i := range i.Metadata.Items {
			s.computeMetadata.Metadata[i.Key] = *i.Value
		}
	}

	return s.computeMetadata, nil
}

// instance returns data from an instance ID using the cache or making a request
func (s *metadataCollector) instance(ctx context.Context, instanceID, zone string) (i *compute.Instance, err error) {
	service, err := compute.NewService(ctx, s.opt...)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting client from Compute service")
	}

	instanceCachedData := s.instanceCache.Get(instanceID)
	if instanceCachedData != nil {
		if computeInstance, ok := instanceCachedData.(*compute.Instance); ok {
			return computeInstance, nil
		}
	}

	instanceData, err := service.Instances.Get(s.projectID, zone, instanceID).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting instance information for instance with ID '%s'", instanceID)
	}
	s.instanceCache.Put(instanceID, instanceData)

	return instanceData, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels[googlecloud.TimeSeriesResponsePathForECSInstanceID]
	}

	return ""
}
