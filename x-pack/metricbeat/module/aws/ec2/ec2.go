// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/pkg/errors"

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
	mb.Registry.MustAddMetricSet("aws", "ec2", New,
		mb.DefaultMetricSet(),
	)
}

// MockCloudWatchClient struct is used for unit tests.
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
	if m.config.Mock == "true" {
		m.MockFetch(report)
		return
	}

	//actual fetch function
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println("Failed to load config: ", err.Error())
	}

	cfg.Region = "us-west-1"
	svcEC2 := ec2.New(cfg)
	//Get a list of regions
	regionsList, err := getRegions(svcEC2)
	if err != nil {
		report.Error(errors.Wrap(err, "getRegions failed"))
	}

	for _, regionName := range regionsList {
		cfg.Region = regionName
		svcEC2 := ec2.New(cfg)
		instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2)
		if err != nil {
			report.Error(errors.Wrap(err, "getInstancesPerRegion failed"))
		}

		svcCloudwatch := cloudwatch.New(cfg)
		for _, instanceID := range instanceIDs {
			init := true
			getMetricDataOutput := cloudwatch.GetMetricDataOutput{NextToken: nil}
			for init || getMetricDataOutput.NextToken != nil {
				init = false
				getMetricDataOutput, err := getMetricDataPerRegion(instanceID, getMetricDataOutput.NextToken, svcCloudwatch)
				if err != nil {
					report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
				}
				reportCloudWatchEvents(getMetricDataOutput, instanceID, instancesOutputs[instanceID], report)
			}
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
		getMetricDataOutput, err := getMetricDataPerRegion(instanceID, nil, svcCloudwatchMock)
		if err != nil {
			report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
		}
		reportCloudWatchEvents(getMetricDataOutput, instanceID, instancesOutputs[instanceID], report)
	}
}

func getRegions(svc ec2iface.EC2API) (regionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	req := svc.DescribeRegionsRequest(input)
	output, err := req.Send()
	if err != nil {
		fmt.Println("Failed DescribeRegions: ", err)
		return
	}
	for _, region := range output.Regions {
		regionsList = append(regionsList, *region.RegionName)
	}
	return
}

func reportCloudWatchEvents(getMetricDataOutput *cloudwatch.GetMetricDataOutput, instanceID string, instanceOutput ec2.Instance, report mb.ReporterV2) {
	machineType, err := instanceOutput.InstanceType.MarshalValue()
	if err != nil {
		report.Error(errors.Wrap(err, "instance.InstanceType.MarshalValue failed"))
	}

	for _, output := range getMetricDataOutput.MetricDataResults {
		if len(output.Values) == 0 {
			break
		}
		event := mb.Event{}
		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", aws.ModuleName)
		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cloud.provider", "aws")
		event.ModuleFields.Put("cloud.instance.id", instanceID)
		event.ModuleFields.Put("cloud.machine.type", machineType)
		event.ModuleFields.Put("cloud.availability_zone", *instanceOutput.Placement.AvailabilityZone)
		event.ModuleFields.Put("cloud.image.id", *instanceOutput.ImageId)

		switch *output.Id {
		case "cpu1":
			event.MetricSetFields = common.MapStr{
				"cpu_utilization": output.Values[0],
			}
		case "cpu2":
			event.MetricSetFields = common.MapStr{
				"cpu_credit_usage": output.Values[0],
			}
		case "cpu3":
			event.MetricSetFields = common.MapStr{
				"cpu_credit_balance": output.Values[0],
			}
		case "cpu4":
			event.MetricSetFields = common.MapStr{
				"cpu_surplus_credit_balance": output.Values[0],
			}
		case "cpu5":
			event.MetricSetFields = common.MapStr{
				"cpu_surplus_credits_charged": output.Values[0],
			}
		case "network1":
			event.MetricSetFields = common.MapStr{
				"network_packets_in": output.Values[0],
			}
		case "network2":
			event.MetricSetFields = common.MapStr{
				"network_packets_out": output.Values[0],
			}
		case "network3":
			event.MetricSetFields = common.MapStr{
				"network_in": output.Values[0],
			}
		case "network4":
			event.MetricSetFields = common.MapStr{
				"network_out": output.Values[0],
			}
		case "disk1":
			event.MetricSetFields = common.MapStr{
				"disk_read_bytes": output.Values[0],
			}
		case "disk2":
			event.MetricSetFields = common.MapStr{
				"disk_write_bytes": output.Values[0],
			}
		case "disk3":
			event.MetricSetFields = common.MapStr{
				"disk_read_ops": output.Values[0],
			}
		case "disk4":
			event.MetricSetFields = common.MapStr{
				"disk_write_ops": output.Values[0],
			}
		case "status1":
			event.MetricSetFields = common.MapStr{
				"status_check_failed": output.Values[0],
			}
		case "status2":
			event.MetricSetFields = common.MapStr{
				"status_check_failed_system": output.Values[0],
			}
		case "status3":
			event.MetricSetFields = common.MapStr{
				"status_check_failed_instance": output.Values[0],
			}
		}
		report.Event(event)
	}
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
			fmt.Println("Error DescribeInstances: ", err)
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

func getMetricDataPerRegion(instanceID string, nextToken *string, svc cloudwatchiface.CloudWatchAPI) (*cloudwatch.GetMetricDataOutput, error) {
	//TODO:remove hard coded variables
	endTime := time.Now()
	duration, err := time.ParseDuration("-10m")
	if err != nil {
		fmt.Println("Error ParseDuration: ", err)
		return nil, err
	}

	startTime := endTime.Add(duration)

	//TODO:add function getMetricNames from environment variables
	dimName := "InstanceId"
	dim := cloudwatch.Dimension{
		Name:  &dimName,
		Value: &instanceID,
	}

	metricDataQuery1 := createMetricDataQuery("cpu1", "CPUUtilization", []cloudwatch.Dimension{dim})
	metricDataQuery2 := createMetricDataQuery("cpu2", "CPUCreditUsage", []cloudwatch.Dimension{dim})
	metricDataQuery3 := createMetricDataQuery("cpu3", "CPUCreditBalance", []cloudwatch.Dimension{dim})
	metricDataQuery4 := createMetricDataQuery("cpu4", "CPUSurplusCreditBalance", []cloudwatch.Dimension{dim})
	metricDataQuery5 := createMetricDataQuery("cpu5", "CPUSurplusCreditsCharged", []cloudwatch.Dimension{dim})
	metricDataQuery6 := createMetricDataQuery("network1", "NetworkPacketsIn", []cloudwatch.Dimension{dim})
	metricDataQuery7 := createMetricDataQuery("network2", "NetworkPacketsOut", []cloudwatch.Dimension{dim})
	metricDataQuery8 := createMetricDataQuery("network3", "NetworkIn", []cloudwatch.Dimension{dim})
	metricDataQuery9 := createMetricDataQuery("network4", "NetworkOut", []cloudwatch.Dimension{dim})
	metricDataQuery10 := createMetricDataQuery("disk1", "DiskReadBytes", []cloudwatch.Dimension{dim})
	metricDataQuery11 := createMetricDataQuery("disk2", "DiskWriteBytes", []cloudwatch.Dimension{dim})
	metricDataQuery12 := createMetricDataQuery("disk3", "DiskReadOps", []cloudwatch.Dimension{dim})
	metricDataQuery13 := createMetricDataQuery("disk4", "DiskWriteOps", []cloudwatch.Dimension{dim})
	metricDataQuery14 := createMetricDataQuery("status1", "StatusCheckFailed", []cloudwatch.Dimension{dim})
	metricDataQuery15 := createMetricDataQuery("status2", "StatusCheckFailed_System", []cloudwatch.Dimension{dim})
	metricDataQuery16 := createMetricDataQuery("status3", "StatusCheckFailed_Instance", []cloudwatch.Dimension{dim})

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
		fmt.Println("GetMetricDataInput Error = ", err.Error())
		return nil, err
	}
	return getMetricDataOutput, nil
}

func createMetricDataQuery(id string, metricName string, dimensions []cloudwatch.Dimension) (metricDataQuery cloudwatch.MetricDataQuery) {
	namespace := "AWS/EC2"
	statistic := "Average"
	// period 5 minutes
	period := int64(300)

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
