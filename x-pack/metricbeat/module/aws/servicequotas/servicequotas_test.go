// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package servicequotas

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/stretchr/testify/assert"
)

// ServiceQuotasClient interface defines the methods used by the MetricSet
type ServiceQuotasClient interface {
	ListServices(ctx context.Context, params *servicequotas.ListServicesInput, optFns ...func(*servicequotas.Options)) (*servicequotas.ListServicesOutput, error)
	ListServiceQuotas(ctx context.Context, params *servicequotas.ListServiceQuotasInput, optFns ...func(*servicequotas.Options)) (*servicequotas.ListServiceQuotasOutput, error)
}

// MockAWSHealthClient implements the ServiceQuotasClient interface
type MockServiceQuotasClient struct{}

func (m *MockServiceQuotasClient) ListServices(ctx context.Context, params *servicequotas.ListServicesInput, optFns ...func(*servicequotas.Options)) (*servicequotas.ListServicesOutput, error) {
	output := &servicequotas.ListServicesOutput{
		NextToken: nil,
		Services: []types.ServiceInfo{
			{
				ServiceCode: aws.String("SERVICE_CODE"),
				ServiceName: aws.String("SERVICE_NAME"),
			},
		},
	}
	return output, nil
}

func (m *MockServiceQuotasClient) ListServiceQuotas(ctx context.Context, params *servicequotas.ListServiceQuotasInput, optFns ...func(*servicequotas.Options)) (*servicequotas.ListServiceQuotasOutput, error) {
	// Mock implementation of ListServiceQuotas method
	output := &servicequotas.ListServiceQuotasOutput{
		Quotas: []types.ServiceQuota{
			{
				Adjustable: true,
				ErrorReason: &types.ErrorReason{ // Mocking nested struct ErrorReason
					ErrorCode:    types.ErrorCodeDependencyThrottlingError,
					ErrorMessage: aws.String("SomeErrorMessage"),
				},
				GlobalQuota: true,
				Period: &types.QuotaPeriod{ // Mocking nested struct QuotaPeriod
					PeriodValue: aws.Int32(3600),
					PeriodUnit:  types.PeriodUnitHour, // Assuming types.PeriodUnitHour is an enum type
				},
				QuotaArn:    aws.String("arn:aws:servicequotas:us-west-2:123456789012:servicequota/service-code/quotaname"),
				QuotaCode:   aws.String("QUOTA_CODE"),
				QuotaName:   aws.String("QUOTA_NAME"),
				ServiceCode: aws.String("SERVICE_CODE"),
				ServiceName: aws.String("SERVICE_NAME"),
				Unit:        aws.String("UNITS"),
				UsageMetric: &types.MetricInfo{ // Mocking nested struct MetricInfo
					MetricName:      aws.String("SomeMetric"),
					MetricNamespace: aws.String("AWS/Service"),
				},
				Value: aws.Float64(100.0),
			},
			// add more mock events as needed
		},
	}
	return output, nil
}

func TestGetEventDetails(t *testing.T) {
	// Mock context
	ctx := context.Background()

	// Create a mock MockServiceQuotas Client
	sqClient := &MockServiceQuotasClient{}
	// Call DescribeEvents

	lsInput := servicequotas.ListServicesInput{
		MaxResults: aws.Int32(10),
	}

	services, err := sqClient.ListServices(ctx, &lsInput)

	assert.NoError(t, err)

	// Validate services.Services is not empty
	assert.NotEmpty(t, services.Services)

	for _, service := range services.Services {
		sqInput := servicequotas.ListServiceQuotasInput{
			ServiceCode: service.ServiceCode,
			MaxResults:  aws.Int32(10),
		}
		quotas, err := sqClient.ListServiceQuotas(ctx, &sqInput)
		assert.NoError(t, err)
		assert.NotEmpty(t, quotas)
	}
}

func TestNormalizePeriodValue(t *testing.T) {
	testCases := []struct {
		name          string
		input         *types.QuotaPeriod
		expectedValue *int32
	}{
		{
			name: "Microsecond",
			input: &types.QuotaPeriod{
				PeriodUnit:  types.PeriodUnitMicrosecond,
				PeriodValue: aws.Int32(1000000), // 1 second in microseconds
			},
			expectedValue: aws.Int32(1),
		},
		{
			name: "Millisecond",
			input: &types.QuotaPeriod{
				PeriodUnit:  types.PeriodUnitMillisecond,
				PeriodValue: aws.Int32(1000), // 1 second in milliseconds
			},
			expectedValue: aws.Int32(1),
		},
		{
			name: "Minute",
			input: &types.QuotaPeriod{
				PeriodUnit:  types.PeriodUnitMinute,
				PeriodValue: aws.Int32(1),
			},
			expectedValue: aws.Int32(60), // 1 minute in seconds
		},
		// Add more test cases for other period units as needed
		{
			name:          "NilInput",
			input:         nil,
			expectedValue: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualValue := normalisePeriodValue(tc.input)
			if !(actualValue == nil && tc.expectedValue == nil) {
				if *actualValue != *tc.expectedValue {
					t.Errorf("Expected period value: %d, got: %d", *tc.expectedValue, *actualValue)
				}
			}
		})
	}
}
