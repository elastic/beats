// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"fmt"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
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

var (
	errOutletClosed = errors.New("input outlet closed")
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// awsCloudWatchInput is a input for AWS CloudWatch logs
type awsCloudWatchInput struct {
	outlet          channel.Outleter // Output of received awscloudwatch logs.
	config          config
	awsConfig       awssdk.Config
	logger          *logp.Logger
	close           chan struct{}
	workerOnce      sync.Once // Guarantees that the worker goroutine is only started once.
	context         *channelContext
	workerWg        sync.WaitGroup // Waits on awscloudwatch worker goroutine.
	stopOnce        sync.Once
	prevCurrentTime time.Time
}

type cwContext struct {
	mux  sync.Mutex
	refs int
	err  error // first error witnessed or multi error
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
func NewInput(cfg *common.Config, connector channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Beta("awsclouwatch input type is used")
	logger := logp.NewLogger(inputName)

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "failed unpacking config")
	}

	logger.Debug("awscloudwatch input config = ", config)
	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: context.DynamicFields,
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

	awsConfig, err := awscommon.GetAWSCredentials(config.AwsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
	}

	closeChannel := make(chan struct{})
	p := &awsCloudWatchInput{
		outlet:    out,
		config:    config,
		awsConfig: awsConfig,
		logger:    logger,
		close:     closeChannel,
		context:   &channelContext{closeChannel},
	}
	return p, nil
}

// Run runs the input
func (p *awsCloudWatchInput) Run() {
	awsConfig := p.awsConfig.Copy()
	awsConfig.Region = p.config.RegionName

	p.workerOnce.Do(func() {
		p.workerWg.Add(1)
		cwConfig := awscommon.EnrichAWSConfigWithEndpoint(p.config.AwsConfig.Endpoint, "cloudwatchlogs", p.config.RegionName, awsConfig)
		p.run(cwConfig)
		p.workerWg.Done()
	})
}

func (p *awsCloudWatchInput) run(cwConfig awssdk.Config) {
	defer p.logger.Infof("awscloudwatch input worker for log group '%v' has stopped.", p.config.LogGroup)
	p.logger.Infof("awscloudwatch input worker for log group: '%v' has started", p.config.LogGroup)

	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	svc := cloudwatchlogs.New(cwConfig)
	prevEndTime := int64(0)
	for p.context.Err() == nil {
		i := 0
		nextToken := ""
		currentTime := time.Now()
		startTime, endTime := getStartPosition(p.config.StartPosition, currentTime, prevEndTime)
		prevEndTime = endTime

		p.logger.Debugf("start_position = %s and startTime = %v", p.config.StartPosition, startTime)

		for nextToken != "" || i == 0 {
			fmt.Println("====== i = ", i)
			fmt.Println("====== nextToken = ", nextToken)

			filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName: awssdk.String(p.config.LogGroup),
				StartTime:    awssdk.Int64(startTime),
				EndTime:      awssdk.Int64(endTime),
				Limit:        awssdk.Int64(p.config.Limit),
			}
			if i != 0 {
				filterLogEventsInput.NextToken = awssdk.String(nextToken)
			}

			req := svc.FilterLogEventsRequest(filterLogEventsInput)
			resp, err := req.Send(ctx)
			if err != nil {
				p.logger.Error("FilterLogEventsRequest failed", err)
				continue
			}

			if resp.NextToken != nil {
				nextToken = *resp.NextToken
			} else {
				nextToken = ""
			}

			logEvents := resp.Events
			fmt.Println("# events = ", len(logEvents))
			p.logger.Debugf("Processing #%v events", len(logEvents))

			errC := make(chan error)
			err = p.processLogEvents(logEvents, errC)
			if err != nil {
				err = errors.Wrap(err, "handleS3Objects failed")
				p.logger.Error(err)
				continue
			}

			// increase counter after making FilterLogEventsRequest API call
			i++
			time.Sleep(time.Duration(200) * time.Millisecond)
		}

		p.logger.Infof("sleeping for %v before checking new logs", p.config.WaitTime)
		time.Sleep(time.Duration(p.config.WaitTime) * time.Second)
		p.logger.Info("done sleeping")
	}
}

func getStartPosition(startPosition string, currentTime time.Time, prevEndTime int64) (startTime int64, endTime int64) {
	switch startPosition {
	case "beginning":
		if prevEndTime != 0 {
			return prevEndTime, int64(currentTime.Nanosecond()) / int64(time.Millisecond)
		}
		return 0, int64(currentTime.Nanosecond()) / int64(time.Millisecond)
	case "end":
		if prevEndTime != 0 {
			return prevEndTime, 0
		}
		return int64(currentTime.Nanosecond()) / int64(time.Millisecond), 0
	}
	return
}

func (p *awsCloudWatchInput) processLogEvents(logEvents []cloudwatchlogs.FilteredLogEvent, errC chan error) error {
	cwCtx := &cwContext{
		refs: 1,
		errC: errC,
	}
	defer cwCtx.done()

	for _, logEvent := range logEvents {
		event := createEvent(logEvent, p.config.LogGroup, p.config.RegionName, cwCtx)
		err := p.forwardEvent(event)
		if err != nil {
			err = errors.Wrap(err, "forwardEvent failed")
			p.logger.Error(err)
			cwCtx.setError(err)
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

func (p *awsCloudWatchInput) forwardEvent(event beat.Event) error {
	ok := p.outlet.OnEvent(event)
	if !ok {
		return errOutletClosed
	}
	return nil
}

// Stop stops the awscloudwatch input
func (p *awsCloudWatchInput) Stop() {
	p.stopOnce.Do(func() {
		defer p.outlet.Close()
		close(p.close)
		p.logger.Info("Stopping awscloudwatch input")
	})
}

// Wait stops the awscloudwatch input.
func (p *awsCloudWatchInput) Wait() {
	p.Stop()
	p.workerWg.Wait()
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
