// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// NewStackdriverCollectorInputData returns a ready to use MetadataCollectorInputData to be sent to Metadata collectors
func NewStackdriverCollectorInputData(ts *monitoringpb.TimeSeries, projectID, zone string, region string, regions []string) *MetadataCollectorInputData {
	return &MetadataCollectorInputData{
		TimeSeries: ts,
		ProjectID:  projectID,
		Zone:       zone,
		Region:     region,
		Regions:    regions,
	}
}

// NewStackdriverMetadataServiceForTimeSeries apart from having a long name takes a time series object to return the
// Stackdriver canonical Metadata extractor
func NewStackdriverMetadataServiceForTimeSeries(ts *monitoringpb.TimeSeries, organizationID, organizationName, projectName string) MetadataService {
	return &StackdriverTimeSeriesMetadataCollector{
		timeSeries:       ts,
		organizationID:   organizationID,
		organizationName: organizationName,
		projectName:      projectName,
	}
}

// StackdriverTimeSeriesMetadataCollector is the implementation of MetadataCollector to collect metrics from Stackdriver
// common TimeSeries objects
type StackdriverTimeSeriesMetadataCollector struct {
	timeSeries       *monitoringpb.TimeSeries
	organizationID   string
	organizationName string
	projectName      string
}

// Metadata parses a Timeseries object to return its metadata divided into "unknown" (first object) and ECS (second
// object https://www.elastic.co/guide/en/ecs/master/index.html)
func (s *StackdriverTimeSeriesMetadataCollector) Metadata(ctx context.Context, in *monitoringpb.TimeSeries) (MetadataCollectorData, error) {
	m := mapstr.M{}

	var availabilityZone, accountID string

	if in.Resource != nil && in.Resource.Labels != nil {
		availabilityZone = in.Resource.Labels[TimeSeriesResponsePathForECSAvailabilityZone]
		accountID = in.Resource.Labels[TimeSeriesResponsePathForECSAccountID]
	}

	ecs := mapstr.M{
		ECSCloud: mapstr.M{
			ECSCloudProject: mapstr.M{
				ECSCloudID:   accountID,
				ECSCloudName: s.projectName,
			},
			ECSCloudProvider: "gcp",
		},
	}
	if s.organizationID != "" {
		_, _ = ecs.Put(ECSCloud+"."+ECSCloudAccount+"."+ECSCloudID, s.organizationID)
	}
	if s.organizationName != "" {
		_, _ = ecs.Put(ECSCloud+"."+ECSCloudAccount+"."+ECSCloudName, s.organizationName)
	}
	if availabilityZone != "" {
		_, _ = ecs.Put(ECSCloud+"."+ECSCloudAvailabilityZone, availabilityZone)

		// Get region name from availability zone name
		region := getRegionName(availabilityZone)
		if region != "" {
			_, _ = ecs.Put(ECSCloud+"."+ECSCloudRegion, region)
		}
	}

	//Remove keys from resource that refers to ECS fields

	if s.timeSeries == nil {
		return MetadataCollectorData{}, fmt.Errorf("no time series data found in google found response")
	}

	if s.timeSeries.Metric != nil {
		metrics := make(map[string]interface{})
		// common.Mapstr seems to not work as expected when deleting keys so I have to iterate over all results to add
		// the ones I want
		for k, v := range s.timeSeries.Metric.Labels {
			if k == TimeSeriesResponsePathForECSInstanceName {
				continue
			}

			metrics[k] = v
		}

		//Do not write metrics labels if it's content is empty
		for k, v := range metrics {
			_, _ = m.Put(LabelMetrics+"."+k, v)
		}
	}

	if s.timeSeries.Resource != nil {
		resources := make(map[string]interface{})
		// common.Mapstr seems to not work as expected when deleting keys so I have to iterate over all results to add
		// the ones I want
		for k, v := range s.timeSeries.Resource.Labels {

			// We are omitting some labels here because they are added separately for services with additional metadata logic.
			// However, we explicitly include the instance_id label to ensure it is not missed for services without additional metadata logic.
			if k == TimeSeriesResponsePathForECSInstanceID {
				_, _ = ecs.Put(ECSCloudInstanceIDKey, v)
				continue
			}

			if k == TimeSeriesResponsePathForECSAvailabilityZone || k == TimeSeriesResponsePathForECSAccountID {
				continue
			}

			resources[k] = v
		}

		//Do not write resources labels if it's content is empty
		for k, v := range resources {
			_, _ = m.Put(LabelResource+"."+k, v)
		}
	}

	if s.timeSeries.Metadata != nil {
		_, _ = m.Put(LabelSystem, s.timeSeries.Metadata.SystemLabels)
		_, _ = m.Put(LabelUserMetadata, s.timeSeries.Metadata.UserLabels)
	}

	return MetadataCollectorData{
		Labels: m,
		ECS:    ecs,
	}, nil
}

func getRegionName(availabilityZone string) string {
	azSplit := strings.Split(availabilityZone, "-")
	if len(azSplit) != 3 {
		return ""
	}

	region := azSplit[0] + "-" + azSplit[1]

	return region
}
