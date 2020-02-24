// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"fmt"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	"time"
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
	outlet     channel.Outleter // Output of received awscloudwatch logs.
	config     config
	awsConfig  awssdk.Config
	logger     *logp.Logger
	close      chan struct{}
	workerOnce sync.Once // Guarantees that the worker goroutine is only started once.
	context    *channelContext
	workerWg   sync.WaitGroup // Waits on awscloudwatch worker goroutine.
	stopOnce   sync.Once
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
	p.workerOnce.Do(func() {

		awsConfig := p.awsConfig.Copy()
		awsConfig.Region = p.config.RegionName

		cwConfig := awscommon.EnrichAWSConfigWithEndpoint(p.config.AwsConfig.Endpoint, "cloudwatchlogs", p.config.RegionName, awsConfig)

		p.workerWg.Add(1)
		go p.run(cwConfig)
		p.workerWg.Done()
	})
}

func (p *awsCloudWatchInput) run(cwConfig  awssdk.Config) {
	defer p.logger.Infof("awscloudwatch input worker for log group '%v' has stopped.", p.config.LogGroup)

	p.logger.Infof("awscloudwatch input worker has started. with log group: %v", p.config.LogGroup)

	ctx, cancelFn := context.WithTimeout(p.context, p.config.APITimeout)
	defer cancelFn()

	svc := cloudwatchlogs.New(cwConfig)
	for p.context.Err() == nil {
		getLogEventsInput := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName: awssdk.String(p.config.LogGroup),
			LogStreamName: awssdk.String(p.config.LogStream),
			Limit: awssdk.Int64(int64(p.config.Limit)),
		}

		req := svc.GetLogEventsRequest(getLogEventsInput)
		resp, err := req.Send(ctx)
		if err != nil {
			fmt.Println("GetLogEventsRequest failed: ", err.Error())
		}

		events := resp.Events
		p.logger.Debugf("Processing #%v events", len(events))
		for i, event := range events {
			f := common.MapStr{
				"message": *event.Message,
				"log": common.MapStr{
					"file.path": p.config.LogGroup + "/" + p.config.LogStream,
				},
				"aws": common.MapStr{
						"log_group": p.config.LogGroup,
						"log_stream":p.config.LogStream,
				},
				"cloud": common.MapStr{
					"provider": "aws",
					"region":   p.config.RegionName,
				},
			}
			beatEvent := beat.Event{
				Timestamp: time.Now(),
				Fields:    f,
				Meta:      common.MapStr{"id": strconv.Itoa(int(*event.Timestamp)) + "-" + strconv.Itoa(int(*event.IngestionTime)) + "-" + strconv.Itoa(i)},
			}
			err = p.forwardEvent(beatEvent)
			if err != nil {
				err = errors.Wrap(err, "forwardEvent failed")
				p.logger.Error(err)
				fmt.Println("forwardEvent failed: ", err.Error())
			}
		}
	}
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
