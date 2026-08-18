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
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v9/libbeat/common/backoff"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(
	ctx context.Context,
	projectID, zone, region string,
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
		instanceCache:    cacheRegistry.Compute,
		logger:           logger.Named("metrics-compute"),
	}

	// Freshen up the cache, later all we have to do is look up the instance
	err := mc.instanceCache.EnsureFresh(func() (map[string]*computepb.Instance, error) {
		instances := make(map[string]*computepb.Instance)
		r := backoff.NewRetryer(3, time.Second, 30*time.Second)

		err := r.Retry(ctx, func() error {
			var err error
			instances, err = mc.fetchComputeInstances(ctx)
			return err
		})

		return instances, err
	})

	return mc, err
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
	projectName      string
	organizationID   string
	organizationName string
	zone             string
	region           string
	regions          []string
	opt              []option.ClientOption
	instanceCache    *gcp.Cache[*computepb.Instance]
	logger           *logp.Logger
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Compute TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	computeMetadata, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceZone(resp))
	if err != nil {
		return gcp.MetadataCollectorData{}, err
	}
	stackdriverLabels := gcp.NewStackdriverMetadataServiceForTimeSeries(resp, s.organizationID, s.organizationName, s.projectName)
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
	computeMetadata := &computeMetadata{
		instanceID: instanceID,
		zone:       zone,
	}

	instance, ok := s.instanceCache.Get(instanceID)
	if !ok {
		s.logger.Warnf("Instance %s not found in compute cache.", instanceID)
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

func (s *metadataCollector) fetchComputeInstances(ctx context.Context) (map[string]*computepb.Instance, error) {
	s.logger.Debug("Executing fetchComputeInstances via CacheRegistry request")

	instancesClient, err := compute.NewInstancesRESTClient(ctx, s.opt...)
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %w", err)
	}
	defer instancesClient.Close()

	start := time.Now()
	s.logger.Debug("Compute API Instances.AggregatedList starting...")

	req := &computepb.AggregatedListInstancesRequest{
		Project: s.projectID,
	}
	it := instancesClient.AggregatedList(ctx, req)
	fetchedInstances := make(map[string]*computepb.Instance)

	pageCount := 0
	instanceCount := 0
	for {
		instancesScopedListPair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			s.logger.Errorf("Error fetching next instance page: %v", err)
			return nil, fmt.Errorf("error iterating compute instances: %w", err)
		}
		pageCount++

		instances := instancesScopedListPair.Value.GetInstances()

		for _, instance := range instances {
			instanceIdStr := strconv.FormatUint(instance.GetId(), 10)
			fetchedInstances[instanceIdStr] = instance
			instanceCount++
		}
	}

	s.logger.Debugf("Compute AggregatedList finished in %s. Fetched %d instances across %d pages.", time.Since(start), instanceCount, pageCount)
	return fetchedInstances, nil
}
