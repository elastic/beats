// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(projectID, zone string, region string, regions []string, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID:        projectID,
		zone:             zone,
		region:           region,
		regions:          regions,
		opt:              opt,
		computeInstances: make(map[uint64]*computepb.Instance),
		logger:           logp.NewLogger("metrics-compute"),
	}, nil
}

// computeMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type computeMetadata struct {
	zone        string
	instanceID  string
	machineType string

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
	projectID        string
	zone             string
	region           string
	regions          []string
	opt              []option.ClientOption
	computeInstances map[uint64]*computepb.Instance
	logger           *logp.Logger
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Compute TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	computeMetadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceZone(resp))
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}
	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp)
	metadataCollectorData, err := stackdriverLabels.Metadata(ctx, resp)
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}

	if resp.Resource != nil && resp.Resource.Labels != nil {
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceIDKey, resp.Resource.Labels[gcp.TimeSeriesResponsePathForECSInstanceID])
	}

	if resp.Metric.Labels != nil {
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceNameKey, resp.Metric.Labels[gcp.TimeSeriesResponsePathForECSInstanceName])
	}

	if computeMetadata.machineType != "" {
		lastIndex := strings.LastIndex(computeMetadata.machineType, "/")
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, computeMetadata.machineType[lastIndex+1:])
	}

	computeMetadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
	computeMetadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

	if computeMetadata.User != nil {
		metadataCollectorData.Labels[gcp.LabelUser] = computeMetadata.User
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
	instance, err := s.instance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error trying to get data from instance '%s' %w", instanceID, err)
	}

	computeMetadata := &computeMetadata{
		instanceID: instanceID,
		zone:       zone,
	}

	if instance == nil {
		s.logger.Debugf("couldn't find instance %s, call Instances.AggregatedList", instanceID)
		return computeMetadata, nil
	}

	labels := instance.GetLabels()
	if labels != nil {
		computeMetadata.User = labels
	}

	machineType := instance.GetMachineType()
	if machineType != "" {
		computeMetadata.machineType = machineType
	}

	metadata := instance.GetMetadata()

	if metadata != nil {
		metadataItems := metadata.GetItems()

		if metadataItems != nil {
			computeMetadata.Metadata = make(map[string]string)

			for _, item := range metadataItems {
				computeMetadata.Metadata[item.GetKey()] = item.GetValue()
			}
		}
	}

	return computeMetadata, nil
}

// instance returns data from an instance ID using the cache or making a request
func (s *metadataCollector) instance(ctx context.Context, instanceID string) (*computepb.Instance, error) {
	s.getComputeInstances(ctx)

	instanceIdInt, _ := strconv.Atoi(instanceID)
	computeInstance, ok := s.computeInstances[uint64(instanceIdInt)]
	if ok {
		return computeInstance, nil
	}

	return nil, nil
}

func (s *metadataCollector) instanceID(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels[gcp.TimeSeriesResponsePathForECSInstanceID]
	}

	return ""
}

func (s *metadataCollector) instanceZone(ts *monitoringpb.TimeSeries) string {
	if ts.Resource != nil && ts.Resource.Labels != nil {
		return ts.Resource.Labels[gcp.TimeSeriesResponsePathForECSAvailabilityZone]
	}

	return ""
}

func (s *metadataCollector) getComputeInstances(ctx context.Context) {
	if len(s.computeInstances) > 0 {
		return
	}

	s.logger.Debug("Compute API Instances.AggregatedList")

	instancesClient, err := compute.NewInstancesRESTClient(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error getting client from compute service: %v", err)
		return
	}

	defer instancesClient.Close()

	it := instancesClient.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{
		Project: s.projectID,
	})

	for {
		instancesScopedListPair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			s.logger.Errorf("error getting next instance from InstancesScopedListPairIterator: %v", err)
			break
		}

		instances := instancesScopedListPair.Value.GetInstances()
		for _, instance := range instances {
			s.computeInstances[instance.GetId()] = instance
		}
	}
}
