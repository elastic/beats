// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"fmt"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	ec2sdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/ec2"
	"github.com/pkg/errors"
	"time"
	"github.com/elastic/beats/libbeat/common"
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
// and aws s3 module metric name, cloudwatch s3 metric name.
var metricIDNameMap1 = map[string][]string{
	"BucketSizeBytes":     {"bucket.size.bytes", "d1"},
	"NumberOfObjects":     {"object.count", "d2"},
}

var metricIDNameMap2 = map[string][]string{
	"d1":     {"bucket.size.bytes", "BucketSizeBytes"},
	"d2":     {"object.count", "NumberOfObjects"},
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

	awsConfig.Region = moduleConfig.DefaultRegion
	svcEC2 := ec2sdk.New(awsConfig)
	regionsList, err := ec2.GetRegions(svcEC2)
	if err != nil {
		err = errors.Wrap(err, "GetRegions failed")
		s3Logger.Error(err.Error())
	}

	// Calculate duration based on period
	durationString, periodSec, err := ec2.ConvertPeriodToDuration(moduleConfig.Period)
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
		regionsList:    regionsList,
		durationString: durationString,
		periodInSec:    periodSec,
		logger:         s3Logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	namespace := "AWS/S3"
	for _, regionName := range m.regionsList {
		m.awsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(*m.awsConfig)
		listMetricsInput := &cloudwatch.ListMetricsInput{Namespace: &namespace}
		reqListMetrics := svcCloudwatch.ListMetricsRequest(listMetricsInput)

		// List metrics of S3 for each region
		listMetricsOutput, err := reqListMetrics.Send()
		if err != nil {
			err = errors.Wrap(err, "ListMetricsRequest failed, skipping region "+regionName)
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		if len(listMetricsOutput.Metrics) == 0 {
			err = errors.Wrap(err, "No S3 buckets in region "+regionName)
			m.logger.Info(err.Error())
			continue
		}

		// GetMetricData for AWS S3 from Cloudwatch
		endTime := time.Now()
		// Testing only
		endTime = endTime.AddDate(0, 0, -3)
		duration, err := time.ParseDuration(m.durationString)
		if err != nil {
			logp.Error(errors.Wrap(err, "Error ParseDuration"))
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}
		startTime := endTime.Add(duration)
		init := true
		getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
		for init || getMetricDataOutput.NextToken != nil {
			init = false
			output, err := getMetricDataPerRegion(listMetricsOutput.Metrics, getMetricDataOutput.NextToken, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "getMetricDataPerRegion failed, skipping region "+regionName)
				m.logger.Error(err.Error())
				report.Error(err)
				continue
			}
			getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
		}
		// Create Cloudwatch Events for S3
		event, info, err := createCloudWatchEvents(getMetricDataOutput.MetricDataResults, regionName)
		if info != "" {
			m.logger.Info(info)
		}

		if err != nil {
			m.logger.Error(err.Error())
			event.Error = err
			report.Event(event)
			continue
		}
		report.Event(event)
	}
}

func getMetricDataPerRegion(listMetricsOutput []cloudwatch.Metric, nextToken *string, svc cloudwatchiface.CloudWatchAPI, startTime time.Time, endTime time.Time) (*cloudwatch.GetMetricDataOutput, error) {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	for _, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(metricIDNameMap1[*listMetric.MetricName][1], listMetric)
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}

	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken: nextToken,
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: metricDataQueries,
	}

	reqGetMetricData := svc.GetMetricDataRequest(getMetricDataInput)
	getMetricDataOutput, err := reqGetMetricData.Send()
	if err != nil {
		logp.Error(errors.Wrap(err, "Error GetMetricDataInput"))
		return nil, err
	}
	return getMetricDataOutput, nil
}

func createMetricDataQuery(id string, metric cloudwatch.Metric) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	// period has to be 1day
	period := int64(86400)
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

func createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, regionName string) (event mb.Event, info string, err error) {
	event.Service = metricsetName
	event.RootFields = common.MapStr{}
	mapOfRootFieldsResults := make(map[string]interface{})
	mapOfRootFieldsResults["service.name"] = metricsetName
	mapOfRootFieldsResults["cloud.provider"] = metricsetName
	mapOfRootFieldsResults["cloud.region"] = regionName

	resultRootFields, err := eventMapping(mapOfRootFieldsResults, schemaRootFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaRootFields in AWS S3 metricbeat module.")
		return
	}
	event.RootFields = resultRootFields

	mapOfMetricSetFieldResults := make(map[string]interface{})
	for _, output := range getMetricDataResults {
		if len(output.Values) == 0 {
			continue
		}
		metricKey := metricIDNameMap2[*output.Id]
		mapOfMetricSetFieldResults[metricKey[0]] = fmt.Sprint(output.Values[0])
	}

	resultMetricSetFields, err := eventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS S3 metricbeat module.")
		return
	}
	event.MetricSetFields = resultMetricSetFields
	return
}
