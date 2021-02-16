// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "resource"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	currentTime := time.Now()
	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		svcResourceAPI := resourcegroupstaggingapi.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "tagging", regionName, awsConfig))

		events, err := m.getResourceEventsPerRegion(svcResourceAPI, regionName, currentTime)
		if err != nil {
			err = fmt.Errorf("getResourceEventsPerRegion failed in region %s: %w", regionName, err)
			m.Logger().Error(err)
			continue
		}

		for _, event := range events {
			if reported := report.Event(event); !reported {
				m.Logger().Debug("Fetch interrupted, failed to emit event")
				return nil
			}
		}
	}
	return nil
}

func (m *MetricSet) getResourceEventsPerRegion(svc resourcegroupstaggingapiiface.ClientAPI, regionName string, currentTime time.Time) ([]mb.Event, error) {
	var events []mb.Event
	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		PaginationToken: nil,
	}
	init := true
	for init || *getResourcesInput.PaginationToken != "" {
		init = false
		getResourcesRequest := svc.GetResourcesRequest(getResourcesInput)
		output, err := getResourcesRequest.Send(context.TODO())
		if err != nil {
			err = errors.Wrap(err, "error GetResources")
			return events, err
		}

		getResourcesInput.PaginationToken = output.PaginationToken
		if len(output.ResourceTagMappingList) == 0 {
			return events, err
		}

		for _, resource := range output.ResourceTagMappingList {
			events = append(events, m.createEvent(resource, regionName, currentTime))
		}
	}
	return events, nil
}

func (m *MetricSet) createEvent(resource resourcegroupstaggingapi.ResourceTagMapping, regionName string, currentTime time.Time) mb.Event {
	// Initialize events and each event only contain one resource
	event := aws.InitEvent(regionName, m.AccountName, m.AccountID, currentTime)
	event.MetricSetFields.Put("arn", *resource.ResourceARN)

	arnParsed, err := arn.Parse(*resource.ResourceARN)
	if err != nil {
		err = fmt.Errorf("arn.Parse failed: %w", *resource.ResourceARN)
		return event
	}

	event.MetricSetFields.Put("name", arnParsed.Resource)
	event.MetricSetFields.Put("service_name", arnParsed.Service)

	// By default, replace dot "." using underscore "_" for tag keys.
	// Note: tag values are not dedotted.
	for _, tag := range resource.Tags {
		event.ModuleFields.Put("tags."+common.DeDot(*tag.Key), *tag.Value)
	}
	return event
}
