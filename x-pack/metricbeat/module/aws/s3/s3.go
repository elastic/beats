// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"fmt"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "s3"

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
	moduleConfig   *aws.Config
	awsConfig      *awssdk.Config
	regionsList    []string
	durationString string
	periodInSec    int
	logger         *logp.Logger
}

// metricIDNameMap is a translating map between createMetricDataQuery id
// and aws ec2 module metric name, cloudwatch ec2 metric name.
var metricIDNameMap = map[string][]string{
	"daily1":     {"bucket.avg.bytes", "BucketSizeBytes"},
	"daily2":     {"object.avg.count", "NumberOfObjects"},
	"request1":     {"request.total.count", "AllRequests"},
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The aws s3 metricset is beta.")
	s3Logger := logp.NewLogger(aws.ModuleName)

	moduleConfig := aws.Config{}
	if err := base.Module().UnpackConfig(&moduleConfig); err != nil {
		return nil, err
	}

	if moduleConfig.Period == "" {
		err := errors.New("period is not set in AWS module config")
		s3Logger.Error(err)
	}

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Get a list of regions
	awsConfig := defaults.Config()
	awsCreds := awssdk.Credentials{
		AccessKeyID:     moduleConfig.AccessKeyID,
		SecretAccessKey: moduleConfig.SecretAccessKey,
	}
	if moduleConfig.SessionToken != "" {
		awsCreds.SessionToken = moduleConfig.SessionToken
	}

	awsConfig.Credentials = awssdk.StaticCredentialsProvider{
		Value: awsCreds,
	}

	awsConfig.Region = moduleConfig.DefaultRegion

	// Calculate duration based on period
	durationString, periodSec, err := convertPeriodToDuration(moduleConfig.Period)
	if err != nil {
		s3Logger.Error(err.Error())
		return nil, err
	}

	// Check if period is set to be multiple of 60s or 300s
	remainder300 := periodSec % 300
	remainder60 := periodSec % 60
	if remainder300 != 0 || remainder60 != 0 {
		err := errors.New("period needs to be set to 60s (or a multiple of 60s) or set to 300s " +
			"(or a multiple of 300s). To avoid data missing or extra costs, please make sure period is set correctly " +
			"in config.yml")
		s3Logger.Info(err)
	}

	return &MetricSet{
		MetricSet:      metricSet,
		moduleConfig:   &moduleConfig,
		awsConfig:      &awsConfig,
		durationString: durationString,
		periodInSec:    periodSec,
		logger:         s3Logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	svcS3 := s3.New(*m.awsConfig)
	listBucketsInput := &s3.ListBucketsInput{}
	req := svcS3.ListBucketsRequest(listBucketsInput)
	output, err := req.Send()
	if err != nil {
		err = errors.Wrap(err, "s3 ListBucketsRequest failed")
		m.logger.Errorf(err.Error())
		report.Error(err)
		return
	}
	bucketNames := []string{}
	for _, name := range output.Buckets {
		bucketNames = append(bucketNames, *name.Name)
	}

	svcCloudwatch := cloudwatch.New(*m.awsConfig)
	fmt.Println("region = ", m.awsConfig.Region)
	for _, bucketName := range bucketNames {
		endTime := time.Now()
		duration, err := time.ParseDuration(m.durationString)
		if err != nil {
			logp.Error(errors.Wrap(err, "Error ParseDuration"))
			return
		}
		startTime := endTime.Add(duration)
		if bucketName == "chow-artifacts" {
			dimName1 := "BucketName"
			dim1 := cloudwatch.Dimension{
				Name:  &dimName1,
				Value: &bucketName,
			}

			dimName2 := "StorageType"
			dimValue2 := "StandardStorage"
			dim2 := cloudwatch.Dimension{
				Name:  &dimName2,
				Value: &dimValue2,
			}

			metricDataQueries := []cloudwatch.MetricDataQuery{}
			for metricID, metricName := range metricIDNameMap {
				metricDataQuery := createMetricDataQuery(metricID, metricName[1], m.periodInSec, []cloudwatch.Dimension{dim1, dim2})
				metricDataQueries = append(metricDataQueries, metricDataQuery)
			}

			getMetricDataInput := &cloudwatch.GetMetricDataInput{
				StartTime:         &startTime,
				EndTime:           &endTime,
				MetricDataQueries: metricDataQueries,
			}
			// fmt.Println("getMetricDataInput = ", getMetricDataInput)

			req := svcCloudwatch.GetMetricDataRequest(getMetricDataInput)
			getMetricDataOutput, err := req.Send()
			if err != nil {
				logp.Error(errors.Wrap(err, "Error GetMetricDataInput"))
				return
			}
			fmt.Println("getMetricDataOutput = ", getMetricDataOutput)
		}
	}

}

func convertPeriodToDuration(period string) (string, int, error) {
	// Amazon EC2 sends metrics to Amazon CloudWatch with 5-minute default frequency.
	// If detailed monitoring is enabled, then data will be available in 1-minute period.
	// Set starttime double the default frequency earlier than the endtime in order to make sure
	// GetMetricDataRequest gets the latest data point for each metric.
	numberPeriod, err := strconv.Atoi(period[0 : len(period)-1])
	if err != nil {
		return "", 0, err
	}

	unitPeriod := period[len(period)-1:]
	switch unitPeriod {
	case "s":
		duration := "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		return duration, numberPeriod, nil
	case "m":
		duration := "-" + strconv.Itoa(numberPeriod*2) + unitPeriod
		periodInSec := numberPeriod * 60
		return duration, periodInSec, nil
	default:
		err = errors.New("invalid period in config. Please reset period in config")
		duration := "-" + strconv.Itoa(numberPeriod*2) + "s"
		return duration, numberPeriod, err
	}
}

func createMetricDataQuery(id string, metricName string, periodInSec int, dimensions []cloudwatch.Dimension) (metricDataQuery cloudwatch.MetricDataQuery) {
	namespace := "AWS/S3"
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
