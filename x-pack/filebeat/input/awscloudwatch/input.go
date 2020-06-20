// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
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

const inputName = "awscloudwatch"

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
	outlet   channel.Outleter // Output of received awscloudwatch logs.
	inputCtx *channelContext

	workerOnce sync.Once      // Guarantees that the worker goroutine is only started once.
	workerWg   sync.WaitGroup // Waits on awscloudwatch worker goroutine.
	stopOnce   sync.Once
	close      chan struct{}

	prevEndTime int64 // track previous endTime for each iteration.
}

type cwContext struct {
	mux  sync.Mutex
	refs int
	err  error
	errC chan error
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

// NewInput creates a new awscloudwatch input
func NewInput(cfg *common.Config, connector channel.Connector, inputContext input.Context) (input.Input, error) {
	cfgwarn.Beta("awsclouwatch input type is used")
	logger := logp.NewLogger(inputName)

	// Extract and validate the input's configuration.
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

	logger.Debug("awscloudwatch input config = ", config)
	awsConfig, err := awscommon.GetAWSCredentials(config.AwsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
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
	in.outlet, err = connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
		ACKEvents: func(privates []interface{}) {
			for _, private := range privates {
				if cwContext, ok := private.(*cwContext); ok {
					cwContext.done()
				}
			}
		},
	})
	if err != nil {
		return nil, err
	}

	in.logger.Info("Initialized AWS CloudWatch input.")
	return in, nil
}

// Run runs the input
func (in *awsCloudWatchInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.logger.Infof("awscloudwatch input worker for log group: '%v' has started", in.config.LogGroup)
			defer in.logger.Infof("awscloudwatch input worker for log group '%v' has stopped.", in.config.LogGroup)
			defer in.workerWg.Done()
			in.run()
		}()
	})
}

func (in *awsCloudWatchInput) run() {
	cwConfig := awscommon.EnrichAWSConfigWithEndpoint(in.config.AwsConfig.Endpoint, "cloudwatchlogs", in.config.RegionName, in.awsConfig)
	svc := cloudwatchlogs.New(cwConfig)
	for in.inputCtx.Err() == nil {
		err := in.getLogEventsFromCloudWatch(svc)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == awssdk.ErrCodeRequestCanceled {
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

func (in *awsCloudWatchInput) getLogEventsFromCloudWatch(svc cloudwatchlogsiface.ClientAPI) error {
	errC := make(chan error, 1)
	cwCtx := &cwContext{
		refs: 1,
		errC: errC,
	}
	defer cwCtx.done()

	ctx, cancelFn := context.WithTimeout(in.inputCtx, in.config.APITimeout)
	defer cancelFn()

	i := 0
	nextToken := ""
	currentTime := time.Now()
	startTime, endTime := getStartPosition(in.config.StartPosition, currentTime, in.prevEndTime)
	in.logger.Debugf("start_position = %s and startTime = %v", in.config.StartPosition, startTime)

	// overwrite prevEndTime using new endTime
	in.prevEndTime = endTime

	for nextToken != "" || i == 0 {
		// construct FilterLogEventsInput
		filterLogEventsInput := constructFilterLogEventsInput(in.config.LogGroup, startTime, endTime, in.config.Limit, i, nextToken)

		// make API request
		req := svc.FilterLogEventsRequest(filterLogEventsInput)
		resp, err := req.Send(ctx)
		if err != nil {
			in.logger.Error("failed FilterLogEventsRequest", err)
			return err
		}

		// get token for next API call
		nextToken = ""
		if resp.NextToken != nil {
			nextToken = *resp.NextToken
		}

		logEvents := resp.Events
		in.logger.Debugf("Processing #%v events", len(logEvents))

		err = in.processLogEvents(logEvents, cwCtx)
		if err != nil {
			err = errors.Wrap(err, "processLogEvents failed")
			in.logger.Error(err)
			cancelFn()
		}

		// increase counter after making FilterLogEventsRequest API call
		i++
	}
	return nil
}

func constructFilterLogEventsInput(logGroup string, startTime int64, endTime int64, limit int64, i int, nextToken string) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(logGroup),
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
		Limit:        awssdk.Int64(limit),
	}
	if i != 0 {
		filterLogEventsInput.NextToken = awssdk.String(nextToken)
	}
	return filterLogEventsInput
}

func getStartPosition(startPosition string, currentTime time.Time, prevEndTime int64) (startTime int64, endTime int64) {
	switch startPosition {
	case "beginning":
		if prevEndTime != int64(0) {
			return prevEndTime, currentTime.UnixNano() / int64(time.Millisecond)
		}
		return 0, currentTime.UnixNano() / int64(time.Millisecond)
	case "end":
		if prevEndTime != int64(0) {
			return prevEndTime, 0
		}
		return currentTime.UnixNano() / int64(time.Millisecond), 0
	}
	return
}

func (in *awsCloudWatchInput) processLogEvents(logEvents []cloudwatchlogs.FilteredLogEvent, cwCtx *cwContext) error {
	for _, logEvent := range logEvents {
		event := createEvent(logEvent, in.config.LogGroup, in.config.RegionName, cwCtx)
		err := in.forwardEvent(event)
		if err != nil {
			err = errors.Wrap(err, "forwardEvent failed")
			in.logger.Error(err)
			cwCtx.setError(err)
			return err
		}
	}
	return nil
}
func createEvent(logEvent cloudwatchlogs.FilteredLogEvent, logGroup string, regionName string, cwCtx *cwContext) beat.Event {
	cwCtx.Inc()

	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": *logEvent.Message,
			"log": common.MapStr{
				"file.path": logGroup + "/" + *logEvent.LogStreamName,
			},
			"aws": common.MapStr{
				"log_group":      logGroup,
				"log_stream":     *logEvent.LogStreamName,
				"ingestion_time": *logEvent.IngestionTime,
				"timestamp":      *logEvent.Timestamp,
				"event_id":       *logEvent.EventId,
			},
			"cloud": common.MapStr{
				"provider": "aws",
				"region":   regionName,
			},
		},
		Private: cwCtx,
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

// Stop stops the awscloudwatch input
func (in *awsCloudWatchInput) Stop() {
	in.stopOnce.Do(func() {
		defer in.outlet.Close()
		close(in.close)
		in.logger.Info("Stopping awscloudwatch input")
	})
}

// Wait is an alias for Stop.
func (in *awsCloudWatchInput) Wait() {
	in.Stop()
	in.workerWg.Wait()
}

func (c *cwContext) setError(err error) {
	// only care about the last error for now
	// TODO: add "Typed" error to error for context
	c.mux.Lock()
	defer c.mux.Unlock()
	c.err = err
}

func (c *cwContext) done() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.refs--
	if c.refs == 0 {
		c.errC <- c.err
		close(c.errC)
	}
}

func (c *cwContext) Inc() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.refs++
}
