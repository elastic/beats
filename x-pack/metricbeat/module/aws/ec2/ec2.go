// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var (
	metricsetName  = "ec2"
	instanceIDIdx  = 0
	metricNameIdx  = 1
	labelSeparator = "|"
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

	// Check if period is set to be multiple of 60s or 300s
	remainder300 := int(metricSet.Period.Seconds()) % 300
	remainder60 := int(metricSet.Period.Seconds()) % 60
	if remainder300 != 0 || remainder60 != 0 {
		err := errors.New("period needs to be set to 60s (or a multiple of 60s) if detailed monitoring is " +
			"enabled for EC2 instances or set to 300s (or a multiple of 300s) if EC2 instances has basic monitoring. " +
			"To avoid data missing or extra costs, please make sure period is set correctly in config.yml")
		base.Logger().Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(m.Period)

	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		svcEC2 := ec2.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "ec2", regionName, awsConfig))

		instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2)
		if err != nil {
			err = errors.Wrap(err, "getInstancesPerRegion failed, skipping region "+regionName)
			m.Logger().Errorf(err.Error())
			report.Error(err)
			continue
		}

		svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "monitoring", regionName, awsConfig))

		namespace := "AWS/EC2"
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
			continue
		}

		var metricDataQueriesTotal []cloudwatch.MetricDataQuery
		for _, instanceID := range instanceIDs {
			metricDataQueriesTotal = append(metricDataQueriesTotal, constructMetricQueries(listMetricsOutput, instanceID, m.Period)...)
		}

		var metricDataOutput []cloudwatch.MetricDataResult
		if len(metricDataQueriesTotal) != 0 {
			// Use metricDataQueries to make GetMetricData API calls
			metricDataOutput, err = aws.GetMetricDataResults(metricDataQueriesTotal, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
				m.Logger().Error(err.Error())
				report.Error(err)
				continue
			}

			// Create Cloudwatch Events for EC2
			events, err := m.createCloudWatchEvents(metricDataOutput, instancesOutputs, regionName)
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
	}

	return nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, instanceID string, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, instanceID, i, period)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func (m *MetricSet) createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, instanceOutput map[string]ec2.Instance, regionName string) (map[string]mb.Event, error) {
	// Initialize events and metricSetFieldResults per instanceID
	events := map[string]mb.Event{}
	metricSetFieldResults := map[string]map[string]interface{}{}
	for instanceID := range instanceOutput {
		events[instanceID] = aws.InitEvent(regionName, m.AccountName, m.AccountID)
		metricSetFieldResults[instanceID] = map[string]interface{}{}
	}

	// monitoring state for each instance
	monitoringStates := map[string]string{}

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(getMetricDataResults)
	if !timestamp.IsZero() {
		for _, output := range getMetricDataResults {
			if len(output.Values) == 0 {
				continue
			}

			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, labelSeparator)
				instanceID := labels[instanceIDIdx]

				// Add tags
				tags := instanceOutput[instanceID].Tags
				if m.TagsFilter != nil {
					// Check with each tag filter
					// If tag filter doesn't exist in tagKeys/tagValues,
					// then do not report this event/instance.
					if exists := aws.CheckTagFiltersExist(m.TagsFilter, tags); !exists {
						delete(events, instanceID)
						continue
					}
				}

				// By default, replace dot "." using under bar "_" for tag keys and values
				for _, tag := range tags {
					events[instanceID].ModuleFields.Put("tags."+common.DeDot(*tag.Key), common.DeDot(*tag.Value))
				}

				machineType, err := instanceOutput[instanceID].InstanceType.MarshalValue()
				if err != nil {
					return events, errors.Wrap(err, "instance.InstanceType.MarshalValue failed")
				}

				events[instanceID].RootFields.Put("cloud.instance.id", instanceID)
				events[instanceID].RootFields.Put("cloud.machine.type", machineType)
				events[instanceID].RootFields.Put("cloud.availability_zone", *instanceOutput[instanceID].Placement.AvailabilityZone)

				if len(output.Values) > timestampIdx {
					metricSetFieldResults[instanceID][labels[metricNameIdx]] = fmt.Sprint(output.Values[timestampIdx])
				}

				instanceStateName, err := instanceOutput[instanceID].State.Name.MarshalValue()
				if err != nil {
					return events, errors.Wrap(err, "instance.State.Name.MarshalValue failed")
				}

				monitoringState, err := instanceOutput[instanceID].Monitoring.State.MarshalValue()
				if err != nil {
					return events, errors.Wrap(err, "instance.Monitoring.State.MarshalValue failed")
				}

				monitoringStates[instanceID] = monitoringState

				events[instanceID].MetricSetFields.Put("instance.image.id", *instanceOutput[instanceID].ImageId)
				events[instanceID].MetricSetFields.Put("instance.state.name", instanceStateName)
				events[instanceID].MetricSetFields.Put("instance.state.code", *instanceOutput[instanceID].State.Code)
				events[instanceID].MetricSetFields.Put("instance.monitoring.state", monitoringState)
				events[instanceID].MetricSetFields.Put("instance.core.count", *instanceOutput[instanceID].CpuOptions.CoreCount)
				events[instanceID].MetricSetFields.Put("instance.threads_per_core", *instanceOutput[instanceID].CpuOptions.ThreadsPerCore)
				publicIP := instanceOutput[instanceID].PublicIpAddress
				if publicIP != nil {
					events[instanceID].MetricSetFields.Put("instance.public.ip", *publicIP)
				}

				events[instanceID].MetricSetFields.Put("instance.public.dns_name", *instanceOutput[instanceID].PublicDnsName)
				events[instanceID].MetricSetFields.Put("instance.private.dns_name", *instanceOutput[instanceID].PrivateDnsName)
				privateIP := instanceOutput[instanceID].PrivateIpAddress
				if privateIP != nil {
					events[instanceID].MetricSetFields.Put("instance.private.ip", *privateIP)
				}
			}
		}
	}

	for instanceID, metricSetFieldsPerInstance := range metricSetFieldResults {
		if len(metricSetFieldsPerInstance) != 0 {
			resultMetricsetFields, err := aws.EventMapping(metricSetFieldsPerInstance, schemaMetricSetFields)
			if err != nil {
				return events, errors.Wrap(err, "EventMapping failed")
			}

			// add rate metrics
			calculateRate(resultMetricsetFields, monitoringStates[instanceID])

			events[instanceID].MetricSetFields.Update(resultMetricsetFields)
			if len(events[instanceID].MetricSetFields) < 5 {
				m.Logger().Info("Missing Cloudwatch data, this is expected for non-running instances" +
					" or a new instance during the first data collection. If this shows up multiple times," +
					" please recheck the period setting in config. Instance ID: " + instanceID)
			}
		}
	}

	return events, nil
}

func calculateRate(resultMetricsetFields common.MapStr, monitoringState string) {
	var period = 300.0
	if monitoringState != "disabled" {
		period = 60.0
	}

	metricList := []string{
		"network.in.bytes",
		"network.out.bytes",
		"network.in.packets",
		"network.out.packets",
		"diskio.read.bytes",
		"diskio.write.bytes",
		"diskio.read.count",
		"diskio.write.count"}

	for _, metricName := range metricList {
		metricValue, err := resultMetricsetFields.GetValue(metricName)
		if err == nil && metricValue != nil {
			rateValue := metricValue.(float64) / period
			resultMetricsetFields.Put(metricName+"_per_sec", rateValue)
		}
	}
}

func getInstancesPerRegion(svc ec2iface.ClientAPI) (instanceIDs []string, instancesOutputs map[string]ec2.Instance, err error) {
	instancesOutputs = map[string]ec2.Instance{}
	output := ec2.DescribeInstancesOutput{NextToken: nil}
	init := true
	for init || output.NextToken != nil {
		init = false
		describeInstanceInput := &ec2.DescribeInstancesInput{}
		req := svc.DescribeInstancesRequest(describeInstanceInput)
		output, err := req.Send(context.Background())
		if err != nil {
			err = errors.Wrap(err, "Error DescribeInstances")
			return nil, nil, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				instanceIDs = append(instanceIDs, *instance.InstanceId)
				instancesOutputs[*instance.InstanceId] = instance
			}
		}
	}
	return
}

func createMetricDataQuery(metric cloudwatch.Metric, instanceID string, index int, period time.Duration) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	periodInSeconds := int64(period.Seconds())
	id := metricsetName + strconv.Itoa(index)
	metricDims := metric.Dimensions

	for _, dim := range metricDims {
		if *dim.Name == "InstanceId" && *dim.Value == instanceID {
			metricName := *metric.MetricName
			label := instanceID + labelSeparator + metricName
			metricDataQuery = cloudwatch.MetricDataQuery{
				Id: &id,
				MetricStat: &cloudwatch.MetricStat{
					Period: &periodInSeconds,
					Stat:   &statistic,
					Metric: &metric,
				},
				Label: &label,
			}
			return
		}
	}
	return
}
