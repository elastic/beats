// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
)

const (
	cacheTTL         = 30 * time.Second
	initialCacheSize = 13
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(projectID, zone string, region string, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID:     projectID,
		zone:          zone,
		region:        region,
		opt:           opt,
		instanceCache: common.NewCache(cacheTTL, initialCacheSize),
		logger:        logp.NewLogger("metrics-compute"),
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
	projectID     string
	zone          string
	region        string
	opt           []option.ClientOption
	instanceCache *common.Cache
	logger        *logp.Logger
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
		metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceIDKey, resp.Resource.Labels[gcp.TimeSeriesResponsePathForECSInstanceID])
	}

	if resp.Metric.Labels != nil {
		metadataCollectorData.ECS.Put(gcp.ECSCloudInstanceNameKey, resp.Metric.Labels[gcp.TimeSeriesResponsePathForECSInstanceName])
	}

	if computeMetadata.machineType != "" {
		lastIndex := strings.LastIndex(computeMetadata.machineType, "/")
		metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, computeMetadata.machineType[lastIndex+1:])
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
		return nil, errors.Wrapf(err, "error trying to get data from instance '%s'", instanceID)
	}

	computeMetadata := &computeMetadata{
		instanceID: instanceID,
		zone:       zone,
	}

	if instance == nil {
		return computeMetadata, nil
	}

	if instance.Labels != nil {
		computeMetadata.User = instance.Labels
	}

	if instance.MachineType != "" {
		computeMetadata.machineType = instance.MachineType
	}

	if instance.Metadata != nil && instance.Metadata.Items != nil {
		computeMetadata.Metadata = make(map[string]string)

		for _, i := range instance.Metadata.Items {
			computeMetadata.Metadata[i.Key] = *i.Value
		}
	}

	return computeMetadata, nil
}

// instance returns data from an instance ID using the cache or making a request
func (s *metadataCollector) instance(ctx context.Context, instanceID string) (*compute.Instance, error) {
	s.refreshInstanceCache(ctx)
	instanceCachedData := s.instanceCache.Get(instanceID)
	if instanceCachedData != nil {
		if computeInstance, ok := instanceCachedData.(*compute.Instance); ok {
			return computeInstance, nil
		}
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

func (s *metadataCollector) refreshInstanceCache(ctx context.Context) {
	// only refresh cache if it is empty
	if s.instanceCache.Size() > 0 {
		return
	}
	s.logger.Debugf("refresh cache with Instances.AggregatedList API")
	computeService, err := compute.NewService(ctx, s.opt...)
	if err != nil {
		s.logger.Errorf("error getting client from Compute service: %v", err)
		return
	}

	req := computeService.Instances.AggregatedList(s.projectID)
	if err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for _, instancesScopedList := range page.Items {
			for _, instance := range instancesScopedList.Instances {
				s.instanceCache.Put(strconv.Itoa(int(instance.Id)), instance)
			}
		}
		return nil
	}); err != nil {
		s.logger.Errorf("google Instances.AggregatedList error: %v", err)
	}
}
