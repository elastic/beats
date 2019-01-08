// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/defaults"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, "ec2", New,
		mb.DefaultMetricSet(),
	)
}

// MockEC2Client struct is used for unit tests.
type MockEC2Client struct {
	ec2iface.EC2API
}

// MockCloudWatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
	config *aws.Config
}

// metricIDNameMap is a translating map between createMetricDataQuery id
// and aws ec2 module metric name.
var metricIDNameMap = map[string]string{
	"cpu1":     "cpu.total.pct",
	"cpu2":     "cpu.credit_usage",
	"cpu3":     "cpu.credit_balance",
	"cpu4":     "cpu.surplus_credit_balance",
	"cpu5":     "cpu.surplus_credits_charged",
	"network1": "network.in.packets",
	"network2": "network.out.packets",
	"network3": "network.in.bytes",
	"network4": "network.out.bytes",
	"disk1":    "diskio.read.bytes",
	"disk2":    "diskio.write.bytes",
	"disk3":    "diskio.read.ops",
	"disk4":    "diskio.write.ops",
	"status1":  "status.check_failed",
	"status2":  "status.check_failed_system",
	"status3":  "status.check_failed_instance",
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The aws ec2 metricset is experimental.")

	config := aws.Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	return &MetricSet{
		MetricSet: metricSet,
		config:    &config,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	//mock Fetch
	if m.config.AwsAccessKeyID == "" || m.config.AwsSecretAccessKey == "" {
		m.MockFetch(report)
		return
	}

	//actual fetch function
	cfg := defaults.Config()
	cfg.Credentials = awssdk.StaticCredentialsProvider{
		Value: awssdk.Credentials{
			AccessKeyID:     m.config.AwsAccessKeyID,
			SecretAccessKey: m.config.AwsSecretAccessKey,
			SessionToken:    m.config.AwsSessionToken,
		},
	}
	cfg.Region = m.config.AwsDefaultRegion

	svcEC2 := ec2.New(cfg)
	//Get a list of regions
	regionsList, err := getRegions(svcEC2)
	if err != nil {
		report.Error(errors.Wrap(err, "getRegions failed"))
		return
	}

	for _, regionName := range regionsList {
		cfg.Region = regionName
		svcEC2 := ec2.New(cfg)
		instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2)
		if err != nil {
			report.Error(errors.Wrap(err, "getInstancesPerRegion failed, skipping region "+regionName))
			continue
		}

		svcCloudwatch := cloudwatch.New(cfg)
		for _, instanceID := range instanceIDs {
			//Calculate duration based on period
			detailedMonitoring := instancesOutputs[instanceID].Monitoring.State
			durationString, periodSec := convertPeriodToDuration(m.config.Period, detailedMonitoring)
			fmt.Println("instanceID = ", instanceID)
			init := true
			getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
			for init || getMetricDataOutput.NextToken != nil {
				init = false
				output, err := getMetricDataPerRegion(durationString, periodSec, instanceID, getMetricDataOutput.NextToken, svcCloudwatch)
				if err != nil {
					report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
					return
				}
				getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
			}
			reportCloudWatchEvents(getMetricDataOutput, instanceID, instancesOutputs[instanceID], regionName, report)
		}
	}
}

// MockFetch methods implements the data gathering and data conversion using MockEC2Client and MockCloudWatchClient
func (m *MetricSet) MockFetch(report mb.ReporterV2) {
	svcEC2Mock := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2Mock)
	if err != nil {
		report.Error(errors.Wrap(err, "getInstancesPerRegion failed"))
	}

	svcCloudwatchMock := &MockCloudWatchClient{}
	for _, instanceID := range instanceIDs {
		detailedMonitoring := instancesOutputs[instanceID].Monitoring.State
		//Calculate duration based on period
		durationString, periodSec := convertPeriodToDuration(m.config.Period, detailedMonitoring)
		fmt.Println("instanceID = ", instanceID)
		getMetricDataOutput, err := getMetricDataPerRegion(durationString, periodSec, instanceID, nil, svcCloudwatchMock)
		if err != nil {
			report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
		}
		reportCloudWatchEvents(getMetricDataOutput, instanceID, instancesOutputs[instanceID], "us-west-1", report)
	}
}

func getRegions(svc ec2iface.EC2API) (regionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	req := svc.DescribeRegionsRequest(input)
	output, err := req.Send()
	if err != nil {
		logp.Error(errors.Wrap(err, "Failed DescribeRegions"))
		return
	}
	for _, region := range output.Regions {
		regionsList = append(regionsList, *region.RegionName)
	}
	return
}

func reportCloudWatchEvents(getMetricDataOutput *cloudwatch.GetMetricDataOutput, instanceID string, instanceOutput ec2.Instance, regionName string, report mb.ReporterV2) {
	machineType, err := instanceOutput.InstanceType.MarshalValue()
	if err != nil {
		report.Error(errors.Wrap(err, "instance.InstanceType.MarshalValue failed"))
	}

	event := mb.Event{}
	event.RootFields = common.MapStr{}
	mapOfRootFieldsResults := make(map[string]interface{})
	mapOfRootFieldsResults["service.name"] = aws.ModuleName
	mapOfRootFieldsResults["cloud.provider"] = "ec2"
	mapOfRootFieldsResults["cloud.instance.id"] = instanceID
	mapOfRootFieldsResults["cloud.machine.type"] = machineType
	mapOfRootFieldsResults["cloud.availability_zone"] = *instanceOutput.Placement.AvailabilityZone
	mapOfRootFieldsResults["cloud.image.id"] = *instanceOutput.ImageId
	mapOfRootFieldsResults["cloud.region"] = regionName

	resultRootFields, err := eventMapping(mapOfRootFieldsResults, schemaRootFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema in AWS EC2 metricbeat module.")
		logp.Error(err)
		report.Error(err)
	}

	mapOfMetricSetFieldResults := make(map[string]interface{})
	for _, output := range getMetricDataOutput.MetricDataResults {
		if len(output.Values) == 0 {
			continue
		}
		metricKey := metricIDNameMap[*output.Id]
		mapOfMetricSetFieldResults[metricKey] = fmt.Sprint(output.Values[0])
	}

	resultMetricSetFields, err := eventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema in AWS EC2 metricbeat module.")
		logp.Error(err)
		report.Error(err)
	}
	report.Event(mb.Event{MetricSetFields: resultMetricSetFields, RootFields: resultRootFields})
}

func getInstancesPerRegion(svc ec2iface.EC2API) (instanceIDs []string, instancesOutputs map[string]ec2.Instance, err error) {
	instancesOutputs = map[string]ec2.Instance{}
	output := ec2.DescribeInstancesOutput{NextToken: nil}
	init := true
	for init || output.NextToken != nil {
		init = false
		describeInstanceInput := &ec2.DescribeInstancesInput{
			Filters: []ec2.Filter{
				{
					Name:   awssdk.String("instance-state-name"),
					Values: []string{"running"},
				},
			},
		}

		req := svc.DescribeInstancesRequest(describeInstanceInput)
		output, err := req.Send()
		if err != nil {
			logp.Error(errors.Wrap(err, "Error DescribeInstances"))
			return nil, nil, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				instanceID := *instance.InstanceId
				instanceIDs = append(instanceIDs, instanceID)
				instancesOutputs[instanceID] = instance
			}
		}
	}
	return
}

func convertPeriodToDuration(period string, detailedMonitoring ec2.MonitoringState) (duration string, periodInSeconds int) {
	// Amazon EC2 sends metrics to Amazon CloudWatch with 5-minute default frequency.
	// If detailed monitoring is enabled, then data will be available in 1-minute period.
	// Set starttime double the default frequency earlier than the endtime in order to make sure
	// GetMetricDataRequest gets the latest data point for each metric.
	numberPeriod, err := strconv.Atoi(period[0 : len(period)-1])
	if err != nil {
		logp.Error(errors.Wrap(err, "Error converting string to int. Use default duration instead."))
		// If failed converting string to int, then set default duration to "-600s" with basic monitoring and "-120s" with
		// detailed monitoring.
		numberPeriod = 300
		if detailedMonitoring == "enabled" {
			numberPeriod = 60
		}
		duration = "-" + strconv.Itoa(numberPeriod*2) + "s"
		periodInSeconds = numberPeriod
		return duration, periodInSeconds
	}

	unitPeriod := period[len(period)-1:]
	// if detailed monitoring is enabled, then period can be larger or equal than 1min.
	// if detailed monitoring is disabled, then period can be larger or equal than 5min.
	switch unitPeriod {
	case "s":
		if detailedMonitoring == "disabled" && numberPeriod < 300 {
			numberPeriod = 300
		} else if detailedMonitoring == "enabled" && numberPeriod < 60 {
			numberPeriod = 60
		}
		duration = "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		periodInSeconds = numberPeriod
	case "m":
		if detailedMonitoring == "disabled" && numberPeriod < 5 {
			numberPeriod = 5
		} else if detailedMonitoring == "enabled" && numberPeriod < 1 {
			numberPeriod = 1
		}
		duration = "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		periodInSeconds = numberPeriod * 60
	default:
		numberPeriod = 300
		if detailedMonitoring == "enabled" {
			numberPeriod = 60
		}
		duration = "-" + strconv.Itoa(numberPeriod*2) + "s"
		periodInSeconds = numberPeriod
	}
	return
}

func getMetricDataPerRegion(durationString string, periodInSec int, instanceID string, nextToken *string, svc cloudwatchiface.CloudWatchAPI) (*cloudwatch.GetMetricDataOutput, error) {
	endTime := time.Now()
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		logp.Error(errors.Wrap(err, "Error ParseDuration"))
		return nil, err
	}

	startTime := endTime.Add(duration)

	dimName := "InstanceId"
	dim := cloudwatch.Dimension{
		Name:  &dimName,
		Value: &instanceID,
	}

	metricDataQuery1 := createMetricDataQuery("cpu1", "CPUUtilization", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery2 := createMetricDataQuery("cpu2", "CPUCreditUsage", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery3 := createMetricDataQuery("cpu3", "CPUCreditBalance", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery4 := createMetricDataQuery("cpu4", "CPUSurplusCreditBalance", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery5 := createMetricDataQuery("cpu5", "CPUSurplusCreditsCharged", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery6 := createMetricDataQuery("network1", "NetworkPacketsIn", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery7 := createMetricDataQuery("network2", "NetworkPacketsOut", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery8 := createMetricDataQuery("network3", "NetworkIn", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery9 := createMetricDataQuery("network4", "NetworkOut", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery10 := createMetricDataQuery("disk1", "DiskReadBytes", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery11 := createMetricDataQuery("disk2", "DiskWriteBytes", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery12 := createMetricDataQuery("disk3", "DiskReadOps", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery13 := createMetricDataQuery("disk4", "DiskWriteOps", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery14 := createMetricDataQuery("status1", "StatusCheckFailed", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery15 := createMetricDataQuery("status2", "StatusCheckFailed_System", periodInSec, []cloudwatch.Dimension{dim})
	metricDataQuery16 := createMetricDataQuery("status3", "StatusCheckFailed_Instance", periodInSec, []cloudwatch.Dimension{dim})

	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken: nextToken,
		StartTime: &startTime,
		EndTime:   &endTime,
		MetricDataQueries: []cloudwatch.MetricDataQuery{metricDataQuery1, metricDataQuery2, metricDataQuery3, metricDataQuery4,
			metricDataQuery5, metricDataQuery6, metricDataQuery7, metricDataQuery8,
			metricDataQuery9, metricDataQuery10, metricDataQuery11, metricDataQuery12,
			metricDataQuery13, metricDataQuery14, metricDataQuery15, metricDataQuery16},
	}

	req := svc.GetMetricDataRequest(getMetricDataInput)
	getMetricDataOutput, err := req.Send()
	if err != nil {
		logp.Error(errors.Wrap(err, "Error GetMetricDataInput"))
		return nil, err
	}
	return getMetricDataOutput, nil
}

func createMetricDataQuery(id string, metricName string, periodInSec int, dimensions []cloudwatch.Dimension) (metricDataQuery cloudwatch.MetricDataQuery) {
	namespace := "AWS/EC2"
	statistic := "Average"
	period := int64(periodInSec)

	metric := cloudwatch.Metric{
		Namespace:  &namespace,
		MetricName: &metricName,
		Dimensions: dimensions,
	}

	metricDataQuery = cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &period,
			Stat:   &statistic,
			Metric: &metric,
		},
	}
	return
}
