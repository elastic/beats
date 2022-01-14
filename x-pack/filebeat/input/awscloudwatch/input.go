// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	inputName = "aws-cloudwatch"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

// awsCloudWatchInput is a input for AWS CloudWatch logs
type awsCloudWatchInput struct {
	config    config
	awsConfig awssdk.Config

	logger   *logp.Logger
	outlet   channel.Outleter // Output of received aws-cloudwatch logs.
	inputCtx *channelContext

	workerOnce sync.Once      // Guarantees that the worker goroutine is only started once.
	workerWg   sync.WaitGroup // Waits on aws-cloudwatch worker goroutine.
	stopOnce   sync.Once
	close      chan struct{}

	prevEndTime int64 // track previous endTime for each iteration.
}

// channelContext implements context.Context by wrapping a channel
type channelContext struct {
	done <-chan struct{}
}

func (c *channelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *channelContext) Done() <-chan struct{}       { return c.done }
func (c *channelContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}
func (c *channelContext) Value(key interface{}) interface{} { return nil }

// NewInput creates a new aws-cloudwatch input
func NewInput(cfg *common.Config, connector channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("aws-clouwatch input type is used")
	logger := logp.NewLogger(inputName)

	// Extract and validate the input's configuration.
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}
	logger.Debug("aws-cloudwatch input config = ", config)

	if config.LogGroupARN != "" {
		logGroupName, regionName, err := parseARN(config.LogGroupARN)
		if err != nil {
			return nil, errors.Wrap(err, "parse log group ARN failed")
		}

		config.LogGroupName = logGroupName
		config.RegionName = regionName
	}

	awsConfig, err := awscommon.InitializeAWSConfig(config.AwsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "InitializeAWSConfig failed")
	}
	awsConfig.Region = config.RegionName

	closeChannel := make(chan struct{})
	in := &awsCloudWatchInput{
		config:      config,
		awsConfig:   awsConfig,
		logger:      logger,
		close:       closeChannel,
		inputCtx:    &channelContext{closeChannel},
		prevEndTime: int64(0),
	}

	// Build outlet for events.
	in.outlet, err = connector.Connect(cfg)
	if err != nil {
		return nil, err
	}

	in.logger.Info("Initialized AWS CloudWatch input.")
	return in, nil
}

// Run runs the input
func (in *awsCloudWatchInput) Run() {
	// Please see https://docs.aws.amazon.com/general/latest/gr/cwl_region.html for more info on Amazon CloudWatch Logs endpoints.
	logsServiceName := awscommon.CreateServiceName("logs", in.config.AwsConfig.FIPSEnabled, in.config.RegionName)
	cwConfig := awscommon.EnrichAWSConfigWithEndpoint(in.config.AwsConfig.Endpoint, logsServiceName, in.config.RegionName, in.awsConfig)
	svc := cloudwatchlogs.New(cwConfig)

	var logGroupNames []string
	var err error
	if in.config.LogGroupNamePrefix != "" {
		logGroupNames, err = in.getLogGroupNames(svc)
		if err != nil {
			in.logger.Error("getLogGroupNames failed: ", err)
			return
		}
	} else {
		logGroupNames = []string{in.config.LogGroupName}
	}

	for _, logGroup := range logGroupNames {
		in.config.LogGroupName = logGroup
		in.workerOnce.Do(func() {
			in.workerWg.Add(1)
			go func() {
				in.logger.Infof("aws-cloudwatch input worker for log group: '%v' has started", in.config.LogGroupName)
				defer in.logger.Infof("aws-cloudwatch input worker for log group '%v' has stopped.", in.config.LogGroupName)
				defer in.workerWg.Done()
				in.run(svc)
			}()
		})
	}
}

func (in *awsCloudWatchInput) run(svc cloudwatchlogsiface.ClientAPI) {
	for in.inputCtx.Err() == nil {
		err := in.getLogEventsFromCloudWatch(svc)
		if err != nil {
			var aerr *awssdk.RequestCanceledError
			if errors.As(err, &aerr) {
				continue
			}
			in.logger.Error("getLogEventsFromCloudWatch failed: ", err)
			continue
		}

		in.logger.Debugf("sleeping for %v before checking new logs", in.config.ScanFrequency)
		time.Sleep(in.config.ScanFrequency)
		in.logger.Debug("done sleeping")
	}
}

func parseARN(logGroupARN string) (string, string, error) {
	arnParsed, err := arn.Parse(logGroupARN)
	if err != nil {
		return "", "", errors.Errorf("error Parse arn %s: %v", logGroupARN, err)
	}

	if strings.Contains(arnParsed.Resource, ":") {
		resourceARNSplit := strings.Split(arnParsed.Resource, ":")
		if len(resourceARNSplit) >= 2 && resourceARNSplit[0] == "log-group" {
			return resourceARNSplit[1], arnParsed.Region, nil
		}
	}
	return "", "", errors.Errorf("cannot get log group name from log group ARN: %s", logGroupARN)
}

// getLogGroupNames uses DescribeLogGroups API to retrieve all log group names
func (in *awsCloudWatchInput) getLogGroupNames(svc cloudwatchlogsiface.ClientAPI) ([]string, error) {
	// construct DescribeLogGroupsInput
	filterLogEventsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: awssdk.String(in.config.LogGroupNamePrefix),
	}

	// make API request
	req := svc.DescribeLogGroupsRequest(filterLogEventsInput)
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(req)
	var logGroupNames []string
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		in.logger.Debugf("Collecting #%v log group names", len(page.LogGroups))
		for _, lg := range page.LogGroups {
			logGroupNames = append(logGroupNames, *lg.LogGroupName)
		}
	}

	if err := p.Err(); err != nil {
		in.logger.Error("failed DescribeLogGroupsRequest: ", err)
		return logGroupNames, err
	}
	return logGroupNames, nil
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (in *awsCloudWatchInput) getLogEventsFromCloudWatch(svc cloudwatchlogsiface.ClientAPI) error {
	currentTime := time.Now()
	startTime, endTime := getStartPosition(in.config.StartPosition, currentTime, in.prevEndTime, in.config.ScanFrequency, in.config.Latency)
	in.logger.Debugf("start_position = %s, startTime = %v, endTime = %v", in.config.StartPosition, time.Unix(startTime/1000, 0), time.Unix(endTime/1000, 0))

	// overwrite prevEndTime using new endTime
	in.prevEndTime = endTime

	// construct FilterLogEventsInput
	filterLogEventsInput := in.constructFilterLogEventsInput(startTime, endTime)

	// make API request
	req := svc.FilterLogEventsRequest(filterLogEventsInput)
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(req)
	for paginator.Next(context.TODO()) {
		page := paginator.CurrentPage()

		logEvents := page.Events
		in.logger.Debugf("Processing #%v events", len(logEvents))
		err := in.processLogEvents(logEvents)
		if err != nil {
			err = errors.Wrap(err, "processLogEvents failed")
			in.logger.Error(err)
		}
	}

	if err := paginator.Err(); err != nil {
		return errors.Wrap(err, "error FilterLogEvents with Paginator")
	}

	// This sleep is to avoid hitting the FilterLogEvents API limit(5 transactions per second (TPS)/account/Region).
	in.logger.Debugf("sleeping for %v before making FilterLogEvents API call again", in.config.APISleep)
	time.Sleep(in.config.APISleep)
	in.logger.Debug("done sleeping")
	return nil
}

func (in *awsCloudWatchInput) constructFilterLogEventsInput(startTime int64, endTime int64) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(in.config.LogGroupName),
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
	}

	if len(in.config.LogStreams) > 0 {
		filterLogEventsInput.LogStreamNames = in.config.LogStreams
	}

	if in.config.LogStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(in.config.LogStreamPrefix)
	}
	return filterLogEventsInput
}

func getStartPosition(startPosition string, currentTime time.Time, prevEndTime int64, scanFrequency time.Duration, latency time.Duration) (startTime int64, endTime int64) {
	if latency != 0 {
		// add latency if config is not 0
		currentTime = currentTime.Add(latency * -1)
	}

	switch startPosition {
	case "beginning":
		if prevEndTime != int64(0) {
			return prevEndTime, currentTime.UnixNano() / int64(time.Millisecond)
		}
		return 0, currentTime.UnixNano() / int64(time.Millisecond)
	case "end":
		if prevEndTime != int64(0) {
			return prevEndTime, currentTime.UnixNano() / int64(time.Millisecond)
		}
		return currentTime.Add(-scanFrequency).UnixNano() / int64(time.Millisecond), currentTime.UnixNano() / int64(time.Millisecond)
	}
	return
}

func (in *awsCloudWatchInput) processLogEvents(logEvents []cloudwatchlogs.FilteredLogEvent) error {
	for _, logEvent := range logEvents {
		event := createEvent(logEvent, in.config.LogGroupName, in.config.RegionName)
		err := in.forwardEvent(event)
		if err != nil {
			err = errors.Wrap(err, "forwardEvent failed")
			in.logger.Error(err)
			return err
		}
	}
	return nil
}

func createEvent(logEvent cloudwatchlogs.FilteredLogEvent, logGroup string, regionName string) beat.Event {
	event := beat.Event{
		Timestamp: time.Unix(*logEvent.Timestamp/1000, 0).UTC(),
		Fields: common.MapStr{
			"message":       *logEvent.Message,
			"log.file.path": logGroup + "/" + *logEvent.LogStreamName,
			"event": common.MapStr{
				"id":       *logEvent.EventId,
				"ingested": time.Now(),
			},
			"awscloudwatch": common.MapStr{
				"log_group":      logGroup,
				"log_stream":     *logEvent.LogStreamName,
				"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
			},
			"cloud": common.MapStr{
				"provider": "aws",
				"region":   regionName,
			},
		},
	}
	event.SetID(*logEvent.EventId)

	return event
}

func (in *awsCloudWatchInput) forwardEvent(event beat.Event) error {
	ok := in.outlet.OnEvent(event)
	if !ok {
		return errors.New("OnEvent returned false. Stopping input worker")
	}
	return nil
}

// Stop stops the aws-cloudwatch input
func (in *awsCloudWatchInput) Stop() {
	in.stopOnce.Do(func() {
		defer in.outlet.Close()
		close(in.close)
		in.logger.Info("Stopping aws-cloudwatch input")
	})
}

// Wait is an alias for Stop.
func (in *awsCloudWatchInput) Wait() {
	in.Stop()
	in.workerWg.Wait()
}
