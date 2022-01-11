// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/go-concert/unison"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	inputName = "aws-cloudwatch"
)

func Plugin(store beater.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from cloudwatch",
		Manager:    &cloudwatchInputManager{store: store},
	}
}

type cloudwatchInputManager struct {
	store beater.StateStore
}

func (im *cloudwatchInputManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (im *cloudwatchInputManager) Create(cfg *common.Config) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.store)
}

// cloudwatchInput is an input for reading logs from CloudWatch periodically.
type cloudwatchInput struct {
	config    config
	awsConfig awssdk.Config
	store     beater.StateStore
}

type cloudwatchPoller struct {
	numberOfWorkers      int
	apiSleep             time.Duration
	region               string
	logStreams           []string
	logStreamPrefix      string
	startTime            int64
	endTime              int64
	prevEndTime          int64
	workerSem            *awscommon.Sem
	log                  *logp.Logger
	store                *statestore.Store
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
}

type logProcessor struct {
	log       *logp.Logger
	publisher beat.Client
	ack       *awscommon.EventACKTracker
}

func newLogProcessor(log *logp.Logger, publisher beat.Client, ctx context.Context) *logProcessor {
	return &logProcessor{
		log:       log,
		publisher: publisher,
		ack:       awscommon.NewEventACKTracker(ctx),
	}
}

func newCloudwatchPoller(log *logp.Logger,
	store *statestore.Store,
	awsRegion string, apiSleep time.Duration,
	numberOfWorkers int, logStreams []string, logStreamPrefix string) *cloudwatchPoller {
	return &cloudwatchPoller{
		numberOfWorkers:      numberOfWorkers,
		apiSleep:             apiSleep,
		region:               awsRegion,
		logStreams:           logStreams,
		logStreamPrefix:      logStreamPrefix,
		startTime:            int64(0),
		endTime:              int64(0),
		prevEndTime:          int64(0),
		workerSem:            awscommon.NewSem(numberOfWorkers),
		log:                  log,
		store:                store,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
	}
}

func newInput(config config, store beater.StateStore) (*cloudwatchInput, error) {
	cfgwarn.Beta("aws-cloudwatch input type is used")
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	if config.LogGroupARN != "" {
		logGroupName, regionName, err := parseARN(config.LogGroupARN)
		if err != nil {
			return nil, errors.Wrap(err, "parse log group ARN failed")
		}

		config.LogGroupName = logGroupName
		config.RegionName = regionName
	}

	awsConfig, err = awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, errors.Wrap(err, "InitializeAWSConfig failed")
	}
	awsConfig.Region = config.RegionName

	return &cloudwatchInput{
		config:    config,
		awsConfig: awsConfig,
		store:     store,
	}, nil
}

func (in *cloudwatchInput) Name() string { return inputName }

func (in *cloudwatchInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *cloudwatchInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	var err error

	persistentStore, err := in.store.Access()
	if err != nil {
		return fmt.Errorf("can not access persistent store: %w", err)
	}

	defer persistentStore.Close()

	// Wrap input Context's cancellation Done channel a context.Context. This
	// goroutine stops with the parent closes the Done channel.
	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Cancelation.Done():
		case <-ctx.Done():
		}
	}()
	defer cancelInputCtx()

	// Create client for publishing events and receive notification of their ACKs.
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   inputContext.Cancelation,
		ACKHandler: awscommon.NewEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	logsServiceName := awscommon.CreateServiceName("logs", in.config.AWSConfig.FIPSEnabled, in.config.RegionName)
	cwConfig := awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, logsServiceName, in.config.RegionName, in.awsConfig)
	svc := cloudwatchlogs.New(cwConfig)

	logGroupNames, err := getLogGroupNames(svc, in.config.LogGroupNamePrefix, in.config.LogGroupName)
	if err != nil {
		return fmt.Errorf("failed to get log group names: %w", err)
	}

	// This loop tries to keep the workers busy as much as possible while
	// honoring the number in config opposed to a simpler loop that does one
	// listing, sequentially processes every object and then does another listing
	logGroupCount := 0
	start := true
	workerWg := new(sync.WaitGroup)
	log := inputContext.Logger.With("aws-cloudwatch")
	cwPoller := newCloudwatchPoller(
		log.Named("cloudwatch_poller"),
		persistentStore,
		in.awsConfig.Region,
		in.config.APISleep,
		in.config.NumberOfWorkers,
		in.config.LogStreams,
		in.config.LogStreamPrefix)
	logProcessor := newLogProcessor(log.Named("log_processor"), client, ctx)

	for ctx.Err() == nil {
		if logGroupCount == 0 {
			currentTime := time.Now()
			cwPoller.startTime, cwPoller.endTime = getStartPosition(in.config.StartPosition, currentTime, cwPoller.prevEndTime, in.config.ScanFrequency, in.config.Latency)
			cwPoller.log.Debugf("start_position = %s, startTime = %v, endTime = %v", in.config.StartPosition, time.Unix(cwPoller.startTime/1000, 0), time.Unix(cwPoller.endTime/1000, 0))
			cwPoller.prevEndTime = cwPoller.endTime
		}

		if start == false {
			cwPoller.log.Debugf("sleeping for %v before checking new logs", in.config.ScanFrequency)
			time.Sleep(in.config.ScanFrequency)
			cwPoller.log.Debug("done sleeping")
		}
		start = false

		// Determine how many workers are available.
		availableWorkers, err := cwPoller.workerSem.AcquireContext(in.config.NumberOfWorkers, ctx)
		if err != nil {
			break
		}

		if availableWorkers == 0 {
			continue
		}

		// Process each log group name asynchronously with a goroutine.
		for i := 0; i < in.config.NumberOfWorkers; i++ {
			if logGroupCount >= len(logGroupNames) {
				// reset logGroupCount
				logGroupCount = 0
				break
			}

			lg := logGroupNames[logGroupCount]
			workerWg.Add(1)
			go func(logGroup string, startTime int64, endTime int64) {
				defer func() {
					cwPoller.log.Infof("aws-cloudwatch input worker for log group '%v' has stopped.", logGroup)
					workerWg.Done()
					cwPoller.workerSem.Release(1)
				}()
				cwPoller.log.Infof("aws-cloudwatch input worker for log group: '%v' has started", logGroup)
				cwPoller.run(svc, logGroup, startTime, endTime, logProcessor)
			}(lg, cwPoller.startTime, cwPoller.endTime)
			logGroupCount++
		}
	}
	// Wait for all workers to finish.
	workerWg.Wait()
	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}

func (p *cloudwatchPoller) run(svc cloudwatchlogsiface.ClientAPI, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) {
	err := p.getLogEventsFromCloudWatch(svc, logGroup, startTime, endTime, logProcessor)
	if err != nil {
		var err *awssdk.RequestCanceledError
		if errors.As(err, &err) {
			p.log.Error("getLogEventsFromCloudWatch failed with RequestCanceledError: ", err)
		}
		p.log.Error("getLogEventsFromCloudWatch failed: ", err)
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
func getLogGroupNames(svc cloudwatchlogsiface.ClientAPI, logGroupNamePrefix string, logGroupName string) ([]string, error) {
	if logGroupNamePrefix == "" {
		return []string{logGroupName}, nil
	}

	// construct DescribeLogGroupsInput
	filterLogEventsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: awssdk.String(logGroupNamePrefix),
	}

	// make API request
	req := svc.DescribeLogGroupsRequest(filterLogEventsInput)
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(req)
	var logGroupNames []string
	for p.Next(context.TODO()) {
		page := p.CurrentPage()
		for _, lg := range page.LogGroups {
			logGroupNames = append(logGroupNames, *lg.LogGroupName)
		}
	}

	if err := p.Err(); err != nil {
		return logGroupNames, err
	}
	return logGroupNames, nil
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (p *cloudwatchPoller) getLogEventsFromCloudWatch(svc cloudwatchlogsiface.ClientAPI, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) error {
	// construct FilterLogEventsInput
	filterLogEventsInput := p.constructFilterLogEventsInput(startTime, endTime, logGroup)

	// make API request
	req := svc.FilterLogEventsRequest(filterLogEventsInput)
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(req)
	for paginator.Next(context.TODO()) {
		page := paginator.CurrentPage()

		logEvents := page.Events
		p.log.Debugf("Processing #%v events", len(logEvents))
		err := logProcessor.processLogEvents(logEvents, logGroup, p.region)
		if err != nil {
			err = errors.Wrap(err, "processLogEvents failed")
			p.log.Error(err)
		}
	}

	if err := paginator.Err(); err != nil {
		return errors.Wrap(err, "error FilterLogEvents with Paginator")
	}

	// This sleep is to avoid hitting the FilterLogEvents API limit(5 transactions per second (TPS)/account/Region).
	p.log.Debugf("sleeping for %v before making FilterLogEvents API call again", p.apiSleep)
	time.Sleep(p.apiSleep)
	p.log.Debug("done sleeping")
	return nil
}

func (p *cloudwatchPoller) constructFilterLogEventsInput(startTime int64, endTime int64, logGroup string) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(logGroup),
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
	}

	if len(p.logStreams) > 0 {
		filterLogEventsInput.LogStreamNames = p.logStreams
	}

	if p.logStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(p.logStreamPrefix)
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

func (p *logProcessor) processLogEvents(logEvents []cloudwatchlogs.FilteredLogEvent, logGroup string, regionName string) error {
	for _, logEvent := range logEvents {
		event := createEvent(logEvent, logGroup, regionName)
		p.publish(p.ack, &event)
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

func (p *logProcessor) publish(ack *awscommon.EventACKTracker, event *beat.Event) {
	ack.Add()
	event.Private = ack
	p.publisher.Publish(*event)
}
