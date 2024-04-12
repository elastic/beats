// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awshealth

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/health"
	"github.com/stretchr/testify/assert"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/health/types"
)

// HealthClient interface defines the methods used by the MetricSet
type HealthClient interface {
	DescribeEvents(ctx context.Context, input *health.DescribeEventsInput, optFns ...func(*health.Options)) (*health.DescribeEventsOutput, error)
	DescribeEventDetails(ctx context.Context, input *health.DescribeEventDetailsInput, optFns ...func(*health.Options)) (*health.DescribeEventDetailsOutput, error)
	DescribeAffectedEntities(ctx context.Context, input *health.DescribeAffectedEntitiesInput, optFns ...func(*health.Options)) (*health.DescribeAffectedEntitiesOutput, error)
}

// MockAWSHealthClient implements the HealthClient interface
type MockAWSHealthClient struct{}

func (m *MockAWSHealthClient) DescribeEvents(ctx context.Context, input *health.DescribeEventsInput, optFns ...func(*health.Options)) (*health.DescribeEventsOutput, error) {
	// Mock implementation of DescribeEvents method
	output := &health.DescribeEventsOutput{
		Events: []types.Event{
			{
				Arn:               aws.String("mock-event-arn-1"),
				EndTime:           aws.Time(time.Now()),
				EventScopeCode:    MapScopeCode("PUBLIC"),
				EventTypeCategory: MapEventTypeCategory("issue"),
				EventTypeCode:     aws.String("mock-event-type-1"),
				LastUpdatedTime:   aws.Time(time.Now()),
				Region:            aws.String("mock-region-1"),
				Service:           aws.String("mock-service-1"),
				StartTime:         aws.Time(time.Now()),
				StatusCode:        MapEventStatusCode("open"),
			},
			// add more mock events as needed
		},
	}
	return output, nil
}

func (m *MockAWSHealthClient) DescribeEventDetails(ctx context.Context, input *health.DescribeEventDetailsInput, optFns ...func(*health.Options)) (*health.DescribeEventDetailsOutput, error) {
	// Mock implementation of DescribeEventDetails method
	ev_desc := "mock-event-description"
	event_arn := "mock-entity-arn-1"
	output := &health.DescribeEventDetailsOutput{
		SuccessfulSet: []types.EventDetails{
			{
				Event: &types.Event{
					Arn: &event_arn,
				},
				EventDescription: &types.EventDescription{
					LatestDescription: &ev_desc,
				},
			},
			// add more successful items as needed
		},
	}
	return output, nil
}

func (m *MockAWSHealthClient) DescribeAffectedEntities(ctx context.Context, input *health.DescribeAffectedEntitiesInput, optFns ...func(*health.Options)) (*health.DescribeAffectedEntitiesOutput, error) {
	// Mock implementation of DescribeAffectedEntities method
	output := &health.DescribeAffectedEntitiesOutput{
		Entities: []types.AffectedEntity{
			{
				AwsAccountId:    aws.String("mock-account-id-1"),
				EntityUrl:       aws.String("mock-entity-url-1"),
				EntityValue:     aws.String("mock-entity-value-1"),
				LastUpdatedTime: aws.Time(time.Now()),
				StatusCode:      MapStatusCode("PENDING"),
				EntityArn:       aws.String("mock-entity-arn-1"),
			},
			// add more affected entities as needed
		},
	}
	return output, nil
}

// ConvertToHealthClient converts MockAWSHealthClient to *health.Client
func (m *MockAWSHealthClient) ConvertToHealthClient() *health.Client {
	return &health.Client{
		// initialize with required options
	}
}

// MapEventStatusCode maps a string status code to its corresponding EventStatusCode enum value
func MapEventStatusCode(eventStatusCode string) types.EventStatusCode {
	switch eventStatusCode {
	case "open":
		return types.EventStatusCodeOpen
	case "closed":
		return types.EventStatusCodeClosed
	default:
		return types.EventStatusCodeUpcoming // Or any default value you prefer
	}
}

// MapEventTypeCategory maps a string status code to its corresponding EventTypeCategory enum value
func MapEventTypeCategory(eventTypeCategory string) types.EventTypeCategory {
	switch eventTypeCategory {
	case "issue":
		return types.EventTypeCategoryIssue
	case "accountNotification":
		return types.EventTypeCategoryAccountNotification
	case "scheduledChange":
		return types.EventTypeCategoryScheduledChange
	default:
		return types.EventTypeCategoryInvestigation // Or any default value you prefer
	}
}

// MapScopeCode maps a string status code to its corresponding EventScopeCode enum value
func MapScopeCode(scopeCode string) types.EventScopeCode {
	switch scopeCode {
	case "PUBLIC":
		return types.EventScopeCodePublic
	case "ACCOUNT_SPECIFIC":
		return types.EventScopeCodeAccountSpecific
	default:
		return types.EventScopeCodeNone // Or any default value you prefer
	}
}

// MapStatusCode maps a string status code to its corresponding EntityStatusCode enum value
func MapStatusCode(statusCode string) types.EntityStatusCode {
	switch statusCode {
	case "PENDING":
		return types.EntityStatusCodeImpaired
	case "RESOLVED":
		return types.EntityStatusCodeUnimpaired
	default:
		return types.EntityStatusCodeUnknown // Or any default value you prefer
	}
}

func TestGetEventDetails(t *testing.T) {
	// Mock context
	ctx := context.Background()

	// Create a mock AWSHealth client
	awsHealth := &MockAWSHealthClient{}
	// Call DescribeEvents
	eventsOutput, err := awsHealth.DescribeEvents(ctx, &health.DescribeEventsInput{})
	assert.NoError(t, err)
	// Validate eventsOutput.Events is not empty
	assert.NotEmpty(t, eventsOutput.Events)

	// Create a slice to store AWSHealthMetrics
	var awsHealthMetrics []AWSHealthMetric

	for _, event := range eventsOutput.Events {
		// Create a new instance of AWSHealthMetric
		awsHealthMetric := AWSHealthMetric{
			EventArn:          *event.Arn,
			EndTime:           *event.EndTime,
			EventScopeCode:    aws.ToString((*string)(&event.EventScopeCode)),
			EventTypeCategory: aws.ToString((*string)(&event.EventTypeCategory)),
			EventTypeCode:     *event.EventTypeCode,
			LastUpdatedTime:   *event.LastUpdatedTime,
			Region:            *event.Region,
			Service:           *event.Service,
			StartTime:         *event.StartTime,
			StatusCode:        aws.ToString((*string)(&event.StatusCode)),
		}
		// Call DescribeEventDetails for the current event
		eventDetailsOutput, err := awsHealth.DescribeEventDetails(ctx, &health.DescribeEventDetailsInput{
			EventArns: []string{*event.Arn},
		})
		assert.NoError(t, err)

		// Validate eventDetailsOutput.SuccessfulSet is not empty
		assert.NotEmpty(t, eventDetailsOutput.SuccessfulSet)

		// Update EventDescription in awsHealthMetric
		if len(eventDetailsOutput.SuccessfulSet) > 0 {
			awsHealthMetric.EventDescription = *eventDetailsOutput.SuccessfulSet[0].EventDescription.LatestDescription
		}

		// Call DescribeAffectedEntities for the current event
		affectedEntitiesOutput, err := awsHealth.DescribeAffectedEntities(ctx, &health.DescribeAffectedEntitiesInput{
			Filter: &types.EntityFilter{
				EventArns: []string{*event.Arn},
			},
		})
		assert.NoError(t, err)

		// Validate affectedEntitiesOutput.Entities is not empty
		assert.NotEmpty(t, affectedEntitiesOutput.Entities)

		// Count affected entities by status
		var pending, resolved, others int32
		for _, entity := range affectedEntitiesOutput.Entities {
			switch aws.ToString((*string)(&entity.StatusCode)) {
			case "PENDING":
				pending++
			case "RESOLVED":
				resolved++
			default:
				others++
			}
			awsHealthMetric.AffectedEntities = append(awsHealthMetric.AffectedEntities,
				AffectedEntityDetails{
					AwsAccountId:    *entity.AwsAccountId,
					EntityUrl:       *entity.EntityUrl,
					EntityValue:     *entity.EntityValue,
					LastUpdatedTime: *entity.LastUpdatedTime,
					StatusCode:      string(entity.StatusCode),
					EntityArn:       *entity.EntityArn,
				},
			)
		}

		// Update affected entities counts in awsHealthMetric
		awsHealthMetric.AffectedEntitiesPending = pending
		awsHealthMetric.AffectedEntitiesResolved = resolved
		awsHealthMetric.AffectedEntitiesOthers = others

		// Append awsHealthMetric to the slice
		awsHealthMetrics = append(awsHealthMetrics, awsHealthMetric)
	}
	for _, metric := range awsHealthMetrics {
		assert.NotEmpty(t, metric.EventArn)
		assert.NotEmpty(t, metric.EventScopeCode)
		assert.NotEmpty(t, metric.EventTypeCategory)
		assert.NotEmpty(t, metric.EventTypeCode)
		assert.NotEmpty(t, metric.Region)
		assert.NotEmpty(t, metric.Service)
		assert.NotEmpty(t, metric.StatusCode)
		assert.NotEmpty(t, metric.LastUpdatedTime)
		assert.NotEmpty(t, metric.StartTime)
		assert.NotEmpty(t, metric.EndTime)
		assert.NotEmpty(t, metric.EventDescription)
		assert.NotEmpty(t, metric.AffectedEntities)
		assert.GreaterOrEqual(t, (metric.AffectedEntitiesOthers + metric.AffectedEntitiesPending + metric.AffectedEntitiesResolved), int32(0))
	}
}
