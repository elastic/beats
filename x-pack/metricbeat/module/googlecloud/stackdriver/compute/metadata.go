// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/x-pack/metricbeat/module/googlecloud"

	"github.com/elastic/beats/libbeat/common"
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(ctx context.Context, projectID, zone string, opt option.ClientOption) (googlecloud.MetadataService, error) {

	_, err := createOrReturnComputeService(ctx, opt)
	if err != nil {
		return nil, err
	}

	return &metadataCollector{
		projectID: projectID,
		zone:      zone,
		opt:       opt,
		instanceCache: struct {
			sync.Mutex
			instances map[string]*compute.Instance
		}{Mutex: sync.Mutex{}, instances: make(map[string]*compute.Instance)},
	}, nil
}

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
	opt       option.ClientOption

	computeMetadata *computeMetadata

	instanceCache struct {
		sync.Mutex
		instances map[string]*compute.Instance
	}
}

// Metadata implements googlecloud.MetadataCollecter to the known set of labels from a Compute TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (common.MapStr, common.MapStr, error) {
	if s.computeMetadata == nil {
		_, err := s.instanceMetadata(ctx, s.instanceID(resp), s.zone)
		if err != nil {
			return nil, nil, err
		}
	}

	stackdriverLabels := googlecloud.NewStackdriverMetadataServiceForTimeSeries(resp)
	output, ecs, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return nil, nil, err
	}

	if resp.Resource != nil && resp.Resource.Labels != nil {
		ecs.Put(googlecloud.ECS_CLOUD+"."+googlecloud.ECS_CLOUD_INSTANCE+"."+googlecloud.ECS_CLOUD_INSTANCE_ID,
			resp.Resource.Labels[googlecloud.JSON_PATH_ECS_INSTANCE_ID])
	}

	if resp.Metric.Labels != nil {
		ecs.Put(googlecloud.ECS_CLOUD+"."+googlecloud.ECS_CLOUD_INSTANCE+"."+googlecloud.ECS_CLOUD_INSTANCE_NAME,
			resp.Metric.Labels[googlecloud.JSON_PATH_ECS_INSTANCE_NAME])
	}

	if s.computeMetadata.machineType != "" {
		ecs.Put(googlecloud.ECS_CLOUD+"."+googlecloud.ECS_CLOUD_MACHINE+"."+googlecloud.ECS_CLOUD_MACHINE_TYPE, s.computeMetadata.machineType)
	}

	s.computeMetadata.Metrics = output[googlecloud.LABEL_METRICS]
	s.computeMetadata.System = output[googlecloud.LABEL_SYSTEM]

	if s.computeMetadata.User != nil {
		output[googlecloud.LABEL_USER] = s.computeMetadata.User
	}

	if s.computeMetadata.Metadata != nil {
		output[googlecloud.LABEL_METADATA] = s.computeMetadata.Metadata
	}

	return output, ecs, nil
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
	service, err := createOrReturnComputeService(ctx, s.opt)
	if err != nil {
		return nil, err
	}

	s.instanceCache.Lock()
	defer s.instanceCache.Unlock()

	if instanceData, ok := s.instanceCache.instances[instanceID]; ok {
		return instanceData, nil
	}

	s.instanceCache.instances[instanceID], err = service.Instances.Get(s.projectID, zone, instanceID).Do()
	if err != nil {
		return nil, err
	}

	return s.instanceCache.instances[instanceID], nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels[googlecloud.JSON_PATH_ECS_INSTANCE_ID]
	}

	return ""
}
