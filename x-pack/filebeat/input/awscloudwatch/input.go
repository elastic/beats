// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "aws-cloudwatch"
)

func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from cloudwatch",
		Manager:    &cloudwatchInputManager{},
	}
}

type cloudwatchInputManager struct {
}

func (im *cloudwatchInputManager) Init(grp unison.Group, mode v2.Mode) error {
	return nil
}

func (im *cloudwatchInputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config)
}

// cloudwatchInput is an input for reading logs from CloudWatch periodically.
type cloudwatchInput struct {
	config    config
	awsConfig awssdk.Config
	metrics   *inputMetrics
}

func newInput(config config) (*cloudwatchInput, error) {
	cfgwarn.Beta("aws-cloudwatch input type is used")
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	if config.LogGroupARN != "" {
		logGroupName, regionName, err := parseARN(config.LogGroupARN)
		if err != nil {
			return nil, fmt.Errorf("parse log group ARN failed: %w", err)
		}

		config.LogGroupName = logGroupName
		config.RegionName = regionName
	}

	if config.RegionName != "" {
		awsConfig.Region = config.RegionName
	}

	return &cloudwatchInput{
		config:    config,
		awsConfig: awsConfig,
	}, nil
}

func (in *cloudwatchInput) Name() string { return inputName }

func (in *cloudwatchInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *cloudwatchInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	var err error

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
		EventListener: awscommon.NewEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	svc := cloudwatchlogs.NewFromConfig(in.awsConfig, func(o *cloudwatchlogs.Options) {
		if in.config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	logGroupNames, err := getLogGroupNames(svc, in.config.LogGroupNamePrefix, in.config.LogGroupName)
	if err != nil {
		return fmt.Errorf("failed to get log group names: %w", err)
	}

	log := inputContext.Logger
	in.metrics = newInputMetrics(inputContext.ID, nil)
	defer in.metrics.Close()
	cwPoller := newCloudwatchPoller(
		log.Named("cloudwatch_poller"),
		in.metrics,
		in.awsConfig.Region,
		in.config)
	logProcessor := newLogProcessor(log.Named("log_processor"), in.metrics, client, ctx)
	cwPoller.metrics.logGroupsTotal.Add(uint64(len(logGroupNames)))
	cwPoller.startWorkers(ctx, svc, logProcessor)
	cwPoller.receive(ctx, logGroupNames, time.Now)
	return nil
}

func parseARN(logGroupARN string) (string, string, error) {
	arnParsed, err := arn.Parse(logGroupARN)
	if err != nil {
		return "", "", fmt.Errorf("error Parse arn %s: %w", logGroupARN, err)
	}

	if strings.Contains(arnParsed.Resource, ":") {
		resourceARNSplit := strings.Split(arnParsed.Resource, ":")
		if len(resourceARNSplit) >= 2 && resourceARNSplit[0] == "log-group" {
			return resourceARNSplit[1], arnParsed.Region, nil
		}
	}
	return "", "", fmt.Errorf("cannot get log group name from log group ARN: %s", logGroupARN)
}

// getLogGroupNames uses DescribeLogGroups API to retrieve all log group names
func getLogGroupNames(svc *cloudwatchlogs.Client, logGroupNamePrefix string, logGroupName string) ([]string, error) {
	if logGroupNamePrefix == "" {
		return []string{logGroupName}, nil
	}

	// construct DescribeLogGroupsInput
	describeLogGroupsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: awssdk.String(logGroupNamePrefix),
	}

	// make API request
	var logGroupNames []string
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, describeLogGroupsInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error DescribeLogGroups with Paginator: %w", err)
		}

		for _, lg := range page.LogGroups {
			logGroupNames = append(logGroupNames, *lg.LogGroupName)
		}
	}
	return logGroupNames, nil
}
