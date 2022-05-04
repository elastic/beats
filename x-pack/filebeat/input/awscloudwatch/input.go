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

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
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

func (im *cloudwatchInputManager) Create(cfg *conf.C) (v2.Input, error) {
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

	log := inputContext.Logger
	metricRegistry := monitoring.GetNamespace("dataset").GetRegistry()
	metrics := newInputMetrics(metricRegistry, inputContext.ID)
	cwPoller := newCloudwatchPoller(
		log.Named("cloudwatch_poller"),
		metrics,
		persistentStore,
		in.awsConfig.Region,
		in.config.APISleep,
		in.config.NumberOfWorkers,
		in.config.LogStreams,
		in.config.LogStreamPrefix)
	logProcessor := newLogProcessor(log.Named("log_processor"), metrics, client, ctx)
	cwPoller.metrics.logGroupsTotal.Add(uint64(len(logGroupNames)))
	return in.Receive(svc, cwPoller, ctx, logProcessor, logGroupNames)
}

func (in *cloudwatchInput) Receive(svc cloudwatchlogsiface.ClientAPI, cwPoller *cloudwatchPoller, ctx context.Context, logProcessor *logProcessor, logGroupNames []string) error {
	// This loop tries to keep the workers busy as much as possible while
	// honoring the number in config opposed to a simpler loop that does one
	// listing, sequentially processes every object and then does another listing
	start := true
	workerWg := new(sync.WaitGroup)
	lastLogGroupOffset := 0
	for ctx.Err() == nil {
		if start == false {
			cwPoller.log.Debugf("sleeping for %v before checking new logs", in.config.ScanFrequency)
			time.Sleep(in.config.ScanFrequency)
			cwPoller.log.Debug("done sleeping")
		}
		start = false

		currentTime := time.Now()
		cwPoller.startTime, cwPoller.endTime = getStartPosition(in.config.StartPosition, currentTime, cwPoller.endTime, in.config.ScanFrequency, in.config.Latency)
		cwPoller.log.Debugf("start_position = %s, startTime = %v, endTime = %v", in.config.StartPosition, time.Unix(cwPoller.startTime/1000, 0), time.Unix(cwPoller.endTime/1000, 0))
		availableWorkers, err := cwPoller.workerSem.AcquireContext(in.config.NumberOfWorkers, ctx)
		if err != nil {
			break
		}

		if availableWorkers == 0 {
			continue
		}

		workerWg.Add(availableWorkers)
		logGroupNamesLength := len(logGroupNames)
		runningGoroutines := 0

		for i := lastLogGroupOffset; i < logGroupNamesLength; i++ {
			if runningGoroutines >= availableWorkers {
				break
			}

			runningGoroutines++
			lastLogGroupOffset = i + 1
			if lastLogGroupOffset >= logGroupNamesLength {
				// release unused workers
				cwPoller.workerSem.Release(availableWorkers - runningGoroutines)
				for j := 0; j < availableWorkers-runningGoroutines; j++ {
					workerWg.Done()
				}
				lastLogGroupOffset = 0
			}

			lg := logGroupNames[i]
			go func(logGroup string, startTime int64, endTime int64) {
				defer func() {
					cwPoller.log.Infof("aws-cloudwatch input worker for log group '%v' has stopped.", logGroup)
					workerWg.Done()
					cwPoller.workerSem.Release(1)
				}()
				cwPoller.log.Infof("aws-cloudwatch input worker for log group: '%v' has started", logGroup)
				cwPoller.run(svc, logGroup, startTime, endTime, logProcessor)
			}(lg, cwPoller.startTime, cwPoller.endTime)
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

func getStartPosition(startPosition string, currentTime time.Time, endTime int64, scanFrequency time.Duration, latency time.Duration) (int64, int64) {
	if latency != 0 {
		// add latency if config is not 0
		currentTime = currentTime.Add(latency * -1)
	}

	switch startPosition {
	case "beginning":
		if endTime != int64(0) {
			return endTime, currentTime.UnixNano() / int64(time.Millisecond)
		}
		return 0, currentTime.UnixNano() / int64(time.Millisecond)
	case "end":
		if endTime != int64(0) {
			return endTime, currentTime.UnixNano() / int64(time.Millisecond)
		}
		return currentTime.Add(-scanFrequency).UnixNano() / int64(time.Millisecond), currentTime.UnixNano() / int64(time.Millisecond)
	}
	return 0, 0
}
