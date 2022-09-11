// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute

import (
	"context"
<<<<<<< HEAD
=======
	"fmt"
	"strconv"
>>>>>>> 3bcefabb28 ([Metricbeat] Add support for multiple regions in GCP (#32964))
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
)

// NewMetadataService returns the specific Metadata service for a GCP Compute resource
func NewMetadataService(projectID, zone string, region string, regions []string, opt ...option.ClientOption) (gcp.MetadataService, error) {
	return &metadataCollector{
		projectID:     projectID,
		zone:          zone,
		region:        region,
		regions:       regions,
		opt:           opt,
		instanceCache: common.NewCache(30*time.Second, 13),
		logger:        logp.NewLogger("metrics-compute"),
	}, nil
}

// computeMetadata is an object to store data in between the extraction and the writing in the destination (to uncouple
// reading and writing in the same method)
type computeMetadata struct {
	// projectID   string
	zone        string
	instanceID  string
	machineType string

	// ts *monitoringpb.TimeSeries

	User     map[string]string
	Metadata map[string]string
	Metrics  interface{}
	System   interface{}
}

type metadataCollector struct {
<<<<<<< HEAD
	projectID string
	zone      string
	region    string
	opt       []option.ClientOption

	computeMetadata *computeMetadata

=======
	projectID     string
	zone          string
	region        string
	regions       []string
	opt           []option.ClientOption
>>>>>>> 3bcefabb28 ([Metricbeat] Add support for multiple regions in GCP (#32964))
	instanceCache *common.Cache
	logger        *logp.Logger
}

// Metadata implements googlecloud.MetadataCollector to the known set of labels from a Compute TimeSeries single point of data.
func (s *metadataCollector) Metadata(ctx context.Context, resp *monitoringpb.TimeSeries) (gcp.MetadataCollectorData, error) {
	// NOTE: ignoring the return value because instanceMetadata changes s.computeMetadata in place.
	// This is probably not thread safe.
	_, err := s.instanceMetadata(ctx, s.instanceID(resp), s.instanceZone(resp))
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

<<<<<<< HEAD
	if s.computeMetadata.machineType != "" {
		lastIndex := strings.LastIndex(s.computeMetadata.machineType, "/")
		metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, s.computeMetadata.machineType[lastIndex+1:])
=======
	if computeMetadata.machineType != "" {
		lastIndex := strings.LastIndex(computeMetadata.machineType, "/")
		_, _ = metadataCollectorData.ECS.Put(gcp.ECSCloudMachineTypeKey, computeMetadata.machineType[lastIndex+1:])
>>>>>>> 3bcefabb28 ([Metricbeat] Add support for multiple regions in GCP (#32964))
	}

	s.computeMetadata.Metrics = metadataCollectorData.Labels[gcp.LabelMetrics]
	s.computeMetadata.System = metadataCollectorData.Labels[gcp.LabelSystem]

	if s.computeMetadata.User != nil {
		metadataCollectorData.Labels[gcp.LabelUser] = s.computeMetadata.User
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
	// FIXME: remove side effect on metadataCollector instance and use return value instead
	i, err := s.instance(ctx, instanceID, zone)
	if err != nil {
<<<<<<< HEAD
		return nil, errors.Wrapf(err, "error trying to get data from instance '%s' in zone '%s'", instanceID, zone)
=======
		return nil, fmt.Errorf("error trying to get data from instance '%s' %w", instanceID, err)
>>>>>>> 3bcefabb28 ([Metricbeat] Add support for multiple regions in GCP (#32964))
	}

	s.computeMetadata = &computeMetadata{
		instanceID: instanceID,
		zone:       zone,
	}

	if i == nil {
		return s.computeMetadata, nil
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
func (s *metadataCollector) instance(ctx context.Context, instanceID, zone string) (*compute.Instance, error) {
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

	if zone != "" {
		instanceData, err := service.Instances.Get(s.projectID, zone, instanceID).Do()
		if err != nil {
			s.logger.Warnf("failed to get instance information for instance '%s' in zone '%s', skipping metadata for instance", instanceID, zone)
			return nil, nil
		}
		s.instanceCache.Put(instanceID, instanceData)
		return instanceData, nil
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
