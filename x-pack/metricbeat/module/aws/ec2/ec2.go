// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
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
	sess, err := session.NewSession()
	if err != nil {
		report.Error(errors.Wrap(err, "Error creating new session"))
	}

	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.StaticProvider{
				Value: credentials.Value{
					AccessKeyID:     m.config.AccessKeyID,
					SecretAccessKey: m.config.SecretAccessKey,
					SessionToken:    m.config.SessionToken,
				},
			},
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{},
			defaults.RemoteCredProvider(*(defaults.Config()), defaults.Handlers()),
		})

	//Get a list of regions
	svcEC2 := ec2.New(sess, &awssdk.Config{
		Region:      awssdk.String("us-west-1"),
		Credentials: creds,
	})
	regionsList, err := getRegions(svcEC2)
	if err != nil {
		report.Error(errors.Wrap(err, "getRegions failed"))
	}

	for _, regionName := range regionsList {
		svcEC2 := ec2.New(sess, &awssdk.Config{
			Region:      &regionName,
			Credentials: creds,
		})
		instanceIDs, err := getInstancesPerRegion(svcEC2)
		if err != nil {
			report.Error(errors.Wrap(err, "getInstancesPerRegion failed"))
		}

		svcCloudwatch := cloudwatch.New(sess, &awssdk.Config{
			Region:      &regionName,
			Credentials: creds,
		})
		for _, instanceID := range instanceIDs {
			init := true
			getMetricDataOutput := cloudwatch.GetMetricDataOutput{NextToken: nil}
			for init || getMetricDataOutput.NextToken != nil {
				init = false
				getMetricDataOutput, err := getMetricDataPerRegion(instanceID, getMetricDataOutput.NextToken, svcCloudwatch)
				if err != nil {
					report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
				}
				reportEvents(getMetricDataOutput, instanceID, report)
			}
		}
	}
}

// MockFetch methods implements the data gathering and data conversion using MockEC2Client and MockCloudWatchClient
func (m *MetricSet) MockFetch(report mb.ReporterV2) {
	svcEC2Mock := &MockEC2Client{}
	instanceIDs, err := getInstancesPerRegion(svcEC2Mock)
	if err != nil {
		report.Error(errors.Wrap(err, "getInstancesPerRegion failed"))
	}

	svcCloudwatchMock := &MockCloudWatchClient{}
	for _, instanceID := range instanceIDs {
		getMetricDataOutput, err := getMetricDataPerRegion(instanceID, nil, svcCloudwatchMock)
		if err != nil {
			report.Error(errors.Wrap(err, "getMetricDataPerRegion failed"))
		}
		reportEvents(getMetricDataOutput, instanceID, report)
	}
}

func getRegions(svc ec2iface.EC2API) (regionsList []string, err error) {
	input := &ec2.DescribeRegionsInput{}
	result, err := svc.DescribeRegions(input)
	if err != nil {
		fmt.Println("Failed DescribeRegions: ", err)
		return
	}

	for _, region := range result.Regions {
		regionsList = append(regionsList, *region.RegionName)
	}
	return
}

func reportEvents(getMetricDataOutput *cloudwatch.GetMetricDataOutput, instanceID string, report mb.ReporterV2) {
	for _, output := range getMetricDataOutput.MetricDataResults {
		if len(output.Values) == 0 {
			break
		}
		switch *output.Id {
		case "cpu1":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":     instanceID,
					"cpu_utilization": *output.Values[0],
				},
			})
		case "cpu2":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":      instanceID,
					"cpu_credit_usage": *output.Values[0],
				},
			})
		case "cpu3":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":        instanceID,
					"cpu_credit_balance": *output.Values[0],
				},
			})
		case "cpu4":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":                instanceID,
					"cpu_surplus_credit_balance": *output.Values[0],
				},
			})
		case "cpu5":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":                 instanceID,
					"cpu_surplus_credits_charged": *output.Values[0],
				},
			})
		case "network1":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":        instanceID,
					"network_packets_in": *output.Values[0],
				},
			})
		case "network2":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"cloud.provider":      "ec2",
					"cloud.instance.id":   instanceID,
					"network_packets_out": *output.Values[0],
				},
			})
		case "network3":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id": instanceID,
					"network_in":  *output.Values[0],
				},
			})
		case "network4":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id": instanceID,
					"network_out": *output.Values[0],
				},
			})
		case "disk1":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":     instanceID,
					"disk_read_bytes": *output.Values[0],
				},
			})
		case "disk2":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":      instanceID,
					"disk_write_bytes": *output.Values[0],
				},
			})
		case "disk3":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":   instanceID,
					"disk_read_ops": *output.Values[0],
				},
			})
		case "disk4":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":    instanceID,
					"disk_write_ops": *output.Values[0],
				},
			})
		case "status1":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":         instanceID,
					"status_check_failed": *output.Values[0],
				},
			})
		case "status2":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":         instanceID,
					"status_check_failed": *output.Values[0],
				},
			})
		case "status3":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":                instanceID,
					"status_check_failed_system": *output.Values[0],
				},
			})
		case "status4":
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"instance_id":                  instanceID,
					"cpu_credit_balance_instances": *output.Values[0],
				},
			})
		}
	}
}

func getInstancesPerRegion(svc ec2iface.EC2API) (instanceIDs []string, err error) {
	describeEC2InstancesOutput := ec2.DescribeInstancesOutput{NextToken: nil}
	init := true
	for init || describeEC2InstancesOutput.NextToken != nil {
		init = false
		describeInstanceInput := &ec2.DescribeInstancesInput{
			NextToken: describeEC2InstancesOutput.NextToken,
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   awssdk.String("instance-state-name"),
					Values: []*string{awssdk.String("running")},
				},
			},
		}
		describeEC2InstancesOutput, err := svc.DescribeInstances(describeInstanceInput)
		if err != nil {
			fmt.Println("Error DescribeInstances: ", err)
			return nil, err
		}

		for _, reservation := range describeEC2InstancesOutput.Reservations {
			for _, instance := range reservation.Instances {
				instanceIDs = append(instanceIDs, *instance.InstanceId)
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
	dimName1 := "InstanceId"
	dim1 := cloudwatch.Dimension{
		Name:  &dimName1,
		Value: &instanceID,
	}

	metricDataQuery1 := createMetricDataQuery("cpu1", "CPUUtilization", []*cloudwatch.Dimension{&dim1})
	metricDataQuery2 := createMetricDataQuery("cpu2", "CPUCreditUsage", []*cloudwatch.Dimension{&dim1})
	metricDataQuery3 := createMetricDataQuery("cpu3", "CPUCreditBalance", []*cloudwatch.Dimension{&dim1})
	metricDataQuery4 := createMetricDataQuery("cpu4", "CPUSurplusCreditBalance", []*cloudwatch.Dimension{&dim1})
	metricDataQuery5 := createMetricDataQuery("cpu5", "CPUSurplusCreditsCharged", []*cloudwatch.Dimension{&dim1})
	metricDataQuery6 := createMetricDataQuery("network1", "NetworkPacketsIn", []*cloudwatch.Dimension{&dim1})
	metricDataQuery7 := createMetricDataQuery("network2", "NetworkPacketsOut", []*cloudwatch.Dimension{&dim1})
	metricDataQuery8 := createMetricDataQuery("network3", "NetworkIn", []*cloudwatch.Dimension{&dim1})
	metricDataQuery9 := createMetricDataQuery("network4", "NetworkOut", []*cloudwatch.Dimension{&dim1})
	metricDataQuery10 := createMetricDataQuery("disk1", "DiskReadBytes", []*cloudwatch.Dimension{&dim1})
	metricDataQuery11 := createMetricDataQuery("disk2", "DiskWriteBytes", []*cloudwatch.Dimension{&dim1})
	metricDataQuery12 := createMetricDataQuery("disk3", "DiskReadOps", []*cloudwatch.Dimension{&dim1})
	metricDataQuery13 := createMetricDataQuery("disk4", "DiskWriteOps", []*cloudwatch.Dimension{&dim1})
	metricDataQuery14 := createMetricDataQuery("status1", "StatusCheckFailed", []*cloudwatch.Dimension{&dim1})
	metricDataQuery15 := createMetricDataQuery("status2", "StatusCheckFailed_System", []*cloudwatch.Dimension{&dim1})
	metricDataQuery16 := createMetricDataQuery("status3", "StatusCheckFailed_Instances", []*cloudwatch.Dimension{&dim1})

	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken: nextToken,
		StartTime: &startTime,
		EndTime:   &endTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{&metricDataQuery1, &metricDataQuery2, &metricDataQuery3, &metricDataQuery4,
			&metricDataQuery5, &metricDataQuery6, &metricDataQuery7, &metricDataQuery8,
			&metricDataQuery9, &metricDataQuery10, &metricDataQuery11, &metricDataQuery12,
			&metricDataQuery13, &metricDataQuery14, &metricDataQuery15, &metricDataQuery16},
	}

	getMetricDataOutput, err := svc.GetMetricData(getMetricDataInput)
	if err != nil {
		fmt.Println("GetMetricDataInput Error = ", err.Error())
		return nil, err
	}
	return getMetricDataOutput, nil
}

func createMetricDataQuery(id string, metricName string, dimensions []*cloudwatch.Dimension) (metricDataQuery cloudwatch.MetricDataQuery) {
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
