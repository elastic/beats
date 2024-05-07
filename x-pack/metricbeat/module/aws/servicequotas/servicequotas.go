// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package servicequotas

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
)

const metricsetName = "servicequotas"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// CostExplorerConfig holds a configuration specific for billing metricset.
type ServiceQuotasConfig struct {
	Servicename []string `config:"service_names"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
	logger              *logp.Logger
	ServiceQuotasConfig ServiceQuotasConfig `config:"service_quotas_config"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logger := logp.NewLogger(metricsetName)
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating aws metricset: %w", err)
	}

	cfgwarn.Beta("The aws:servicequota metricset is beta.")

	config := struct {
		ServiceQuotasConfig ServiceQuotasConfig `config:"service_quotas_config"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet:           metricSet,
		logger:              logger,
		ServiceQuotasConfig: config.ServiceQuotasConfig,
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

	sqClient := servicequotas.NewFromConfig(
		awsConfig,
		func(o *servicequotas.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		},
	)
	// // Create a map to store the mapping between ServiceName and ServiceCode
	// serviceMap := make(map[string]string)

	// Get the Service code and Service Name details
	serviceDetails := m.GetServiceDetails(ctx, sqClient)

	// Filter service details based on configuration
	serviceMap := m.filterServiceDetails(serviceDetails)

	// Fetch service quotas and create events
	events := m.fetchServiceQuotas(ctx, sqClient, serviceMap)

	// // If m.ServiceQuotasConfig.Servicename is empty, include all elements of servDets in serviceMap
	// if len(m.ServiceQuotasConfig.Servicename) == 0 {
	// 	for _, sd := range servDets {
	// 		serviceMap[*sd.ServiceName] = *sd.ServiceCode
	// 	}
	// } else {
	// 	// Include only the elements of servDets that match the values in m.ServiceQuotasConfig.Servicename
	// 	for _, sd := range servDets {
	// 		serviceName := *sd.ServiceName
	// 		if contains(m.ServiceQuotasConfig.Servicename, serviceName) {
	// 			serviceMap[serviceName] = *sd.ServiceCode
	// 		}
	// 	}
	// }
	// var events []mb.Event
	// for _, serviceCode := range serviceMap {
	// 	qts := m.GetServiceQuotaDetails(ctx, sqClient, serviceCode)
	// 	for _, qt := range qts {
	// 		event := mb.Event{
	// 			MetricSetFields: mapstr.M{
	// 				"adjustable":   qt.Adjustable,
	// 				"global_quota": qt.GlobalQuota,
	// 				"quota_arn":    awssdk.ToString(qt.QuotaArn),
	// 				"quota_code":   awssdk.ToString(qt.QuotaCode),
	// 				"quota_name":   awssdk.ToString(qt.QuotaName),
	// 				"service_name": awssdk.ToString(qt.ServiceName),
	// 				"unit":         awssdk.ToString(qt.Unit),
	// 				"value":        awssdk.ToFloat64(qt.Value),
	// 			},
	// 			RootFields: mapstr.M{
	// 				"cloud.provider": "aws",
	// 			},
	// 			Service: "aws-health",
	// 		}

	// 		// fmt.Println("-----------------------------------------------")
	// 		// fmt.Printf("Adjustable - %t\n", qt.Adjustable)

	// 		if qt.ErrorReason != nil && qt.ErrorReason.ErrorMessage != nil {
	// 			event.MetricSetFields["error_reason"] = awssdk.ToString(qt.ErrorReason.ErrorMessage)
	// 			// fmt.Printf("Error Reason - %s\n", awssdk.ToString(qt.ErrorReason.ErrorMessage))
	// 		}

	// 		// fmt.Printf("Global Quota - %t\n", qt.GlobalQuota)

	// 		if qt.Period != nil {
	// 			periodValue := normalisePeriodValue(qt.Period)
	// 			event.MetricSetFields["period_value"] = *periodValue
	// 			event.MetricSetFields["period_unit"] = types.PeriodUnitSecond
	// 		}
	// 		// if qt.QuotaArn != nil {
	// 		// 	fmt.Printf("QuotaArn - %s\n", awssdk.ToString(qt.QuotaArn))
	// 		// }
	// 		// if qt.QuotaCode != nil {
	// 		// 	fmt.Printf("Quota Code - %s\n", awssdk.ToString(qt.QuotaCode))
	// 		// }
	// 		// if qt.QuotaName != nil {
	// 		// 	fmt.Printf("Quota Name - %s\n", awssdk.ToString(qt.QuotaName))
	// 		// }
	// 		// if qt.ServiceName != nil {
	// 		// 	fmt.Printf("Service Name - %s\n", awssdk.ToString(qt.ServiceName))
	// 		// }
	// 		// if qt.Unit != nil {
	// 		// 	fmt.Printf("Unit - %s\n", awssdk.ToString(qt.Unit))
	// 		// }
	// 		// if qt.Value != nil {
	// 		// 	fmt.Printf("Usage Value - %f\n", awssdk.ToFloat64(qt.Value))
	// 		// }
	// 		//fmt.Println("-----------------------------------------------")
	// 		events = append(events, event)

	// 	}
	// 	//fmt.Printf("%s - %s\n", serviceCode, serviceName)
	// }
	for _, event := range events {
		report.Event(event)
	}
	return nil
}

// filterServiceDetails filters service details based on configuration
func (m *MetricSet) filterServiceDetails(serviceDetails []types.ServiceInfo) map[string]string {
	serviceMap := make(map[string]string)
	for _, sd := range serviceDetails {
		serviceName := *sd.ServiceName
		if len(m.ServiceQuotasConfig.Servicename) == 0 || contains(m.ServiceQuotasConfig.Servicename, serviceName) {
			serviceMap[serviceName] = *sd.ServiceCode
		}
	}
	return serviceMap
}

// normalisePeriodValue normalizes the period value based on the period unit.
func normalisePeriodValue(qt *types.QuotaPeriod) *int32 {
	if qt == nil {
		return nil
	}
	periodValue := int32(0)

	switch qt.PeriodUnit {
	case types.PeriodUnitMicrosecond:
		periodValue = awssdk.ToInt32(qt.PeriodValue) / 1000000
	case types.PeriodUnitMillisecond:
		periodValue = awssdk.ToInt32(qt.PeriodValue) / 1000
	case types.PeriodUnitMinute:
		periodValue = awssdk.ToInt32(qt.PeriodValue) * 60
	case types.PeriodUnitHour:
		periodValue = awssdk.ToInt32(qt.PeriodValue) * 60 * 60
	case types.PeriodUnitDay:
		periodValue = awssdk.ToInt32(qt.PeriodValue) * 60 * 60 * 24
	case types.PeriodUnitWeek:
		periodValue = awssdk.ToInt32(qt.PeriodValue) * 60 * 60 * 24 * 7
	case types.PeriodUnitSecond:
		periodValue = awssdk.ToInt32(qt.PeriodValue)
	default:
		return nil
	}
	return &periodValue
}

// contains checks if a string exists in a slice of strings and returns true if found, otherwise false.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// GetServiceQuotaDetails retrieves the applied quota value for the specified AWS service
func (m *MetricSet) GetServiceQuotaDetails(ctx context.Context, sqClient *servicequotas.Client, scode string) []types.ServiceQuota {

	var (
		quotasTemp []types.ServiceQuota
		quotas     []types.ServiceQuota
	)
	// Construct input parameters for listing service quotas
	sqInput := servicequotas.ListServiceQuotasInput{
		ServiceCode: &scode,
		MaxResults:  awssdk.Int32(10),
	}
	// Create a paginator for listing service quotas
	sqPages := servicequotas.NewListServiceQuotasPaginator(
		sqClient,
		&sqInput,
	)
	// Iterate through each page of results
	for sqPages.HasMorePages() {
		currentPage, err := sqPages.NextPage(ctx)
		if err != nil {
			// Log any errors and break the loop
			m.Logger().Errorf("[AWS ServiceQuota] ListServiceQuotas failed with : %w", err)
			break
		}
		quotasTemp = currentPage.Quotas
		quotas = append(quotas, quotasTemp...)
	}
	// Return the final slice containing all quotas
	return quotas

}

// fetchServiceQuotas fetches service quotas and creates events
func (m *MetricSet) fetchServiceQuotas(ctx context.Context, sqClient *servicequotas.Client, serviceMap map[string]string) []mb.Event {
	var events []mb.Event
	for _, serviceCode := range serviceMap {
		quotas := m.GetServiceQuotaDetails(ctx, sqClient, serviceCode)
		for _, qt := range quotas {
			event := m.createEvent(qt)
			events = append(events, event)
		}
	}
	return events
}

// GetServiceDetails retrieves the names of services
func (m *MetricSet) GetServiceDetails(ctx context.Context, sqClient *servicequotas.Client) []types.ServiceInfo {

	var (
		servDetails     []types.ServiceInfo
		servDetailsTemp []types.ServiceInfo
	)
	// Call ListServices API
	lsInput := servicequotas.ListServicesInput{
		MaxResults: awssdk.Int32(10),
	}
	svPages := servicequotas.NewListServicesPaginator(
		sqClient,
		&lsInput,
	)
	for svPages.HasMorePages() {
		// Iterate through each page of results
		currentPage, err := svPages.NextPage(ctx)
		if err != nil {
			// Log any errors and break the loop
			m.Logger().Errorf("[AWS ServiceQuota] ListServices failed with : %w", err)
			break
		}
		// Append the services from the current page to the list of service details
		servDetailsTemp = currentPage.Services
		servDetails = append(servDetails, servDetailsTemp...)
	}
	return servDetails
}

// createEvent creates a metricbeat event from service quota details
func (m *MetricSet) createEvent(qt types.ServiceQuota) mb.Event {
	event := mb.Event{
		MetricSetFields: mapstr.M{
			"adjustable":   qt.Adjustable,
			"global_quota": qt.GlobalQuota,
			"quota_arn":    awssdk.ToString(qt.QuotaArn),
			"quota_code":   awssdk.ToString(qt.QuotaCode),
			"quota_name":   awssdk.ToString(qt.QuotaName),
			"service_name": awssdk.ToString(qt.ServiceName),
			"unit":         awssdk.ToString(qt.Unit),
			"value":        awssdk.ToFloat64(qt.Value),
		},
		RootFields: mapstr.M{
			"cloud.provider": "aws",
		},
		Service: "aws-servicequotas",
	}

	if qt.ErrorReason != nil && qt.ErrorReason.ErrorMessage != nil {
		event.MetricSetFields["error_reason"] = awssdk.ToString(qt.ErrorReason.ErrorMessage)
	}

	if qt.Period != nil {
		periodValue := normalisePeriodValue(qt.Period)
		event.MetricSetFields["period_value"] = *periodValue
		event.MetricSetFields["period_unit"] = types.PeriodUnitSecond
	}

	return event
}
