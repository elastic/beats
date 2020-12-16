// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package resource

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "resource"
)

// ResourceNameARN contains resource name and ARN
type ResourceNameARN struct {
	name string
	arn  string
}

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
	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		svcResourceAPI := resourcegroupstaggingapi.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "tagging", regionName, awsConfig))

		resources, err := getResourcesPerRegion(svcResourceAPI)
		if err != nil {
			err = errors.Wrap(err, "getResourcesPerRegion failed, skipping region "+regionName)
			m.Logger().Errorf(err.Error())
			report.Error(err)
			continue
		}

		// Create Cloudwatch Events for EC2
		events, err := m.createEvents(resources, regionName)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		for _, event := range events {
			if len(event.MetricSetFields) != 0 {
				if reported := report.Event(event); !reported {
					m.Logger().Debug("Fetch interrupted, failed to emit event")
					return nil
				}
			}
		}
	}
	return nil
}

func getResourcesPerRegion(svc resourcegroupstaggingapiiface.ClientAPI) ([]ResourceNameARN, error) {
	var resources []ResourceNameARN
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
			return nil, err
		}

		getResourcesInput.PaginationToken = output.PaginationToken
		if len(output.ResourceTagMappingList) == 0 {
			return nil, err
		}

		for _, resource := range output.ResourceTagMappingList {
			var discoveredResource ResourceNameARN
			serviceName, err := findServiceNameFromARN(*resource.ResourceARN)
			if err != nil {
				err = errors.Wrap(err, "error findServiceNameFromARN")
				return nil, err
			}
			discoveredResource.name = serviceName
			discoveredResource.arn = *resource.ResourceARN
			resources = append(resources, discoveredResource)
		}
	}
	return resources, nil
}

func findServiceNameFromARN(resourceARN string) (string, error) {
	arnParsed, err := arn.Parse(resourceARN)
	if err != nil {
		err = errors.Wrap(err, "error Parse arn")
		return "", err
	}

	return arnParsed.Service, nil
}

func (m *MetricSet) createEvents(resources []ResourceNameARN, regionName string) (map[string]mb.Event, error) {
	// Initialize events and each event only contain one resource
	events := map[string]mb.Event{}
	for _, resource := range resources {
		identifier := regionName + m.AccountID + resource.arn
		if _, ok := events[identifier]; !ok {
			events[identifier] = aws.InitEvent(regionName, m.AccountName, m.AccountID, time.Now())
		}
		events[identifier].MetricSetFields.Put("resource.name", resource.name)
		events[identifier].MetricSetFields.Put("resource.arn", resource.arn)
	}
	return events, nil
}
