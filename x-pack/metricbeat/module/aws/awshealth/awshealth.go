// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awshealth

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/health"
	"github.com/aws/aws-sdk-go-v2/service/health/types"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const metricsetName = "awshealth"

var (
	locale = "en"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

type AffectedEntityDetails struct {
	AwsAccountId    string    `json:"aws_account_id"`
	EntityUrl       string    `json:"entity_url"`
	EntityValue     string    `json:"entity_value"`
	LastUpdatedTime time.Time `json:"last_updated_time"`
	StatusCode      string    `json:"status_code"`
	EntityArn       string    `json:"entity_arn"`
}

type AWSHealthMetric struct {
	EventArn                 string                  `json:"event_arn"`
	EndTime                  time.Time               `json:"end_time"`
	EventScopeCode           string                  `json:"event_scope_code"`
	EventTypeCategory        string                  `json:"event_type_category"`
	EventTypeCode            string                  `json:"event_type_code"`
	LastUpdatedTime          time.Time               `json:"last_updated_time"`
	Region                   string                  `json:"region"`
	Service                  string                  `json:"service"`
	StartTime                time.Time               `json:"start_time"`
	StatusCode               string                  `json:"status_code"`
	AffectedEntitiesPending  int32                   `json:"affected_entities_pending"`
	AffectedEntitiesResolved int32                   `json:"affected_entities_resolved"`
	AffectedEntitiesOthers   int32                   `json:"affected_entities_others"`
	AffectedEntities         []AffectedEntityDetails `json:"affected_entities"`
	EventDescription         string                  `json:"event_description"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
	logger *logp.Logger
	Config Config `config:"aws_health_config"`
}

// Config holds the configuration specific for aws-awshealth metricset
type Config struct {
	EventARNPattern []string `config:"event_arns_pattern"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logger := logp.NewLogger(metricsetName)
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating aws metricset: %w", err)
	}

	cfgwarn.Beta("The aws:awshealth metricset is beta.")

	config := struct {
		Config Config `config:"aws_health_config"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    logger,
		Config:    config.Config,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, report mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var config aws.Config
	if err := m.Module().UnpackConfig(&config); err != nil {
		return err
	}

	awsConfig := m.MetricSet.AwsConfig.Copy()

	health_client := health.NewFromConfig(awsConfig, func(o *health.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})
	events := m.getEventDetails(ctx, health_client)
	for _, event := range events {
		report.Event(event)
	}

	return nil
}

// getEventDetails retrieves AWS health events and their details using the provided AWS Health client.
// It returns a list of Metricbeat events containing relevant AWS health information.
func (m *MetricSet) getEventDetails(
	ctx context.Context,
	awsHealth *health.Client,
) []mb.Event {
	// Define event filter to fetch only upcoming and open events
	eventFilter := types.EventFilter{
		EventStatusCodes: []types.EventStatusCode{
			types.EventStatusCodeUpcoming,
			types.EventStatusCodeOpen,
		},
	}
	var (
		deEvents          []types.Event
		affPage           health.DescribeAffectedEntitiesPaginator
		healthDetails     []AWSHealthMetric
		healthDetailsTemp []AWSHealthMetric
		affEntityTemp     AffectedEntityDetails
		affInputParams    health.DescribeAffectedEntitiesInput
	)

	// Create an instance of DescribeEventsInput with desired parameters
	deInputParams := health.DescribeEventsInput{
		Filter: &eventFilter,
	}

	// Define options for DescribeEventsPaginator
	deOptions := &health.DescribeEventAggregatesPaginatorOptions{
		Limit:                10,
		StopOnDuplicateToken: true,
	}

	// Function option to apply options to the paginator
	deOptFn := func(options *health.DescribeEventsPaginatorOptions) {
		// Apply the provided options
		options.Limit = deOptions.Limit
		options.StopOnDuplicateToken = deOptions.StopOnDuplicateToken
	}
	// Define options for DescribeAffectedEntitiesPaginator
	affOptions := &health.DescribeAffectedEntitiesPaginatorOptions{
		Limit:                10,
		StopOnDuplicateToken: true,
	}
	// Function option to apply options to the paginator
	affOptFn := func(options *health.DescribeAffectedEntitiesPaginatorOptions) {
		// Apply the provided options
		options.Limit = affOptions.Limit
		options.StopOnDuplicateToken = affOptions.StopOnDuplicateToken
	}
	// Create DescribeEventsPaginator with AWS Health client and options
	dePage := health.NewDescribeEventsPaginator(awsHealth, &deInputParams, deOptFn)

	for dePage.HasMorePages() {
		healthDetailsTemp = []AWSHealthMetric{}

		// Perform actions for the current page
		currentPage, err := dePage.NextPage(ctx)
		if err != nil {
			m.Logger().Errorf("[AWS Health] DescribeEvents failed with : %w", err)
			break
		}
		deEvents = currentPage.Events
		eventArns := make([]string, len(deEvents))
		// Iterate through events to extract relevant information
		for i, de := range deEvents {
			healthDetailsTemp = append(healthDetailsTemp, AWSHealthMetric{
				EventArn:          awssdk.ToString(de.Arn),
				EndTime:           awssdk.ToTime(de.EndTime),
				EventScopeCode:    string(de.EventScopeCode),
				EventTypeCategory: string(de.EventTypeCategory),
				EventTypeCode:     awssdk.ToString(de.EventTypeCode),
				LastUpdatedTime:   awssdk.ToTime(de.LastUpdatedTime),
				Region:            awssdk.ToString(de.Region),
				Service:           awssdk.ToString(de.Service),
				StartTime:         awssdk.ToTime(de.StartTime),
				StatusCode:        string(de.StatusCode),
			})
			eventArns[i] = awssdk.ToString(de.Arn)
		}
		// Fetch event details for the current page of events
		eventDetails, err := awsHealth.DescribeEventDetails(ctx, &health.DescribeEventDetailsInput{
			EventArns: eventArns,
			Locale:    &locale,
		})
		if err != nil {
			m.Logger().Errorf("[AWS Health] DescribeEventDetails failed with : %w", err)
			break
		}
		// Fetch event description for the current page of events
		successSet := eventDetails.SuccessfulSet
		for x := range successSet {
			for y := range healthDetailsTemp {
				if awssdk.ToString(successSet[x].Event.Arn) == healthDetailsTemp[y].EventArn {
					healthDetailsTemp[y].EventDescription = awssdk.ToString(successSet[x].EventDescription.LatestDescription)
				}
			}
		}
		// Fetch affected entities related to event ARNs in the current page
		affInputParams = health.DescribeAffectedEntitiesInput{
			Filter: &types.EntityFilter{
				EventArns: eventArns,
			},
		}
		affPage = *health.NewDescribeAffectedEntitiesPaginator(
			awsHealth,
			&affInputParams,
			affOptFn,
		)

		for affPage.HasMorePages() {
			// Fetch current page of affected entities
			affCurrentPage, err := affPage.NextPage(ctx)
			if err != nil {
				m.Logger().Errorf("[AWS Health] DescribeAffectedEntitie failed with : %w", err)
				break
			}
			// Extract relevant details of affected entities and match them with event details
			for _, ace := range affCurrentPage.Entities {
				affEntityTemp = AffectedEntityDetails{
					AwsAccountId:    awssdk.ToString(ace.AwsAccountId),
					EntityUrl:       awssdk.ToString(ace.EntityUrl),
					EntityValue:     awssdk.ToString(ace.EntityValue),
					LastUpdatedTime: awssdk.ToTime(ace.LastUpdatedTime),
					StatusCode:      string(ace.StatusCode),
					EntityArn:       awssdk.ToString(ace.EntityArn),
				}
				for l, hd := range healthDetailsTemp {
					if awssdk.ToString(ace.EventArn) == hd.EventArn {
						healthDetailsTemp[l].AffectedEntities = append(healthDetailsTemp[l].AffectedEntities, affEntityTemp)
						switch string(ace.StatusCode) {
						case "PENDING":
							healthDetailsTemp[l].AffectedEntitiesPending++
						case "RESOLVED":
							healthDetailsTemp[l].AffectedEntitiesResolved++
						case "":
							// Do Nothing
						default:
							healthDetailsTemp[l].AffectedEntitiesOthers++

						}
					}
				}
			}
		}
		// Append current page's health details to the overall list
		healthDetails = append(healthDetails, healthDetailsTemp...)
	}
	// Convert health details to Metricbeat events
	var events = make([]mb.Event, 0, len(healthDetails))
	for _, detail := range healthDetails {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"event_arn":                  detail.EventArn,
				"end_time":                   detail.EndTime,
				"event_scope_code":           detail.EventScopeCode,
				"event_type_category":        detail.EventTypeCategory,
				"event_type_code":            detail.EventTypeCode,
				"last_updated_time":          detail.LastUpdatedTime,
				"region":                     detail.Region,
				"service":                    detail.Service,
				"start_time":                 detail.StartTime,
				"status_code":                detail.StatusCode,
				"affected_entities":          detail.AffectedEntities,
				"event_description":          detail.EventDescription,
				"affected_entities_pending":  detail.AffectedEntitiesPending,
				"affected_entities_resolved": detail.AffectedEntitiesResolved,
				"affected_entities_others":   detail.AffectedEntitiesOthers,
			},
			RootFields: mapstr.M{
				"cloud.provider": "aws",
			},
			Service: "aws-health",
		}
		events = append(events, event)
	}
	return events
}
