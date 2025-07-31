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
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "aws-cloudwatch"
)

func Plugin(logger *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from cloudwatch",
		Manager:    &cloudwatchInputManager{store: store, logger: logger},
	}
}

type cloudwatchInputManager struct {
	store  statestore.States
	logger *logp.Logger
}

func (im *cloudwatchInputManager) Init(grp unison.Group) error {
	return nil
}

func (im *cloudwatchInputManager) Create(cfg *conf.C) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config, im.store, im.logger)
}

// cloudwatchInput is an input for reading logs from CloudWatch periodically.
type cloudwatchInput struct {
	config    config
	awsConfig awssdk.Config
	store     statestore.States
	metrics   *inputMetrics
	status    status.StatusReporter
}

func newInput(config config, store statestore.States, logger *logp.Logger) (*cloudwatchInput, error) {
	logger.Warn(cfgwarn.Beta("aws-cloudwatch input type is used"))

	// perform AWS configuration validation
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	return &cloudwatchInput{
		config:    config,
		store:     store,
		awsConfig: awsConfig,
	}, nil
}

func (in *cloudwatchInput) Name() string { return inputName }

func (in *cloudwatchInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *cloudwatchInput) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)
	log := inputContext.Logger

	// setup status reporter
	in.status = inputContext.StatusReporter
	if in.status == nil {
		in.status = &debugCWStatusReporter{log}
	}

	defer in.status.UpdateStatus(status.Stopped, "Input stopped")
	in.status.UpdateStatus(status.Starting, "Input starting")

	handler, err := newStateHandler(log, in.config, in.store)
	if err != nil {
		in.status.UpdateStatus(status.Failed, "State registry creation failure")
		return fmt.Errorf("failed to create state handler: %w", err)
	}
	defer handler.Close()

	in.status.UpdateStatus(status.Configuring, "Configuring input")
	var logGroupIDs []string
	logGroupIDs, region, err := fromConfig(in.config, in.awsConfig)
	if err != nil {
		in.status.UpdateStatus(status.Failed, "Configuration loading error")
		return fmt.Errorf("error processing configurations: %w", err)
	}

	in.awsConfig.Region = region
	svc := cloudwatchlogs.NewFromConfig(in.awsConfig, func(o *cloudwatchlogs.Options) {
		if in.config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	if len(logGroupIDs) == 0 {
		// We haven't extracted group identifiers directly from the input configurations,
		// now fallback to provided LogGroupNamePrefix and use derived service client to derive logGroupIDs
		logGroupIDs, err = getLogGroupNames(svc, in.config.LogGroupNamePrefix, in.config.IncludeLinkedAccountsForPrefixMode)
		if err != nil {
			in.status.UpdateStatus(status.Failed, "Configuration loading error")
			return fmt.Errorf("failed to get log group names from LogGroupNamePrefix: %w", err)
		}
	}

	in.metrics = newInputMetrics(inputContext.ID, nil)
	defer in.metrics.Close()
	cwPoller := newCloudwatchPoller(
		log.Named("cloudwatch_poller"),
		in.metrics,
		region,
		in.config,
		handler,
		in.status)

	in.status.UpdateStatus(status.Running, "Input is running")

	cwPoller.metrics.logGroupsTotal.Add(uint64(len(logGroupIDs)))
	err = cwPoller.startWorkers(ctx, svc, pipeline)
	if err != nil {
		in.status.UpdateStatus(status.Failed, "Error starting input processors")
		return err
	}

	log.Debugf("Config latency = %f", cwPoller.config.Latency)
	log.Debugf("Config scan_frequency = %f", cwPoller.config.ScanFrequency)
	log.Debugf("Config api_sleep = %f", cwPoller.config.APISleep)
	cwPoller.receive(ctx, logGroupIDs, time.Now)
	return nil
}

// debugCWStatusReporter with debugging logs. This is typically used when running in standalone mode.
type debugCWStatusReporter struct {
	log *logp.Logger
}

func (n *debugCWStatusReporter) UpdateStatus(status status.Status, msg string) {
	n.log.Debugf("CloudWatch input status updated: status: %s, message: %s", status, msg)
}

// fromConfig is a helper to parse input configurations and derive logGroupIDs & aws region
// Returned logGroupIDs could be empty, which require other fallback mechanisms to derive them.
// See getLogGroupNames for example.
func fromConfig(cfg config, awsCfg awssdk.Config) (logGroupIDs []string, region string, err error) {
	// LogGroupARN has precedence over LogGroupName & RegionName
	if cfg.LogGroupARN != "" {
		parsedArn, err := arn.Parse(cfg.LogGroupARN)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse log group ARN: %w", err)
		}

		if parsedArn.Region == "" {
			return nil, "", fmt.Errorf("failed to parse log group ARN: missing region")
		}

		// refine to match AWS API parameter regex of logGroupIdentifier
		groupId := strings.TrimSuffix(cfg.LogGroupARN, ":*")
		logGroupIDs = append(logGroupIDs, groupId)

		return logGroupIDs, parsedArn.Region, nil
	}

	// then fallback to LogrGroupName
	if cfg.LogGroupName != "" {
		logGroupIDs = append(logGroupIDs, cfg.LogGroupName)
	}

	// finally derive region
	if cfg.RegionName != "" {
		region = cfg.RegionName
	} else {
		region = awsCfg.Region
	}

	return logGroupIDs, region, nil
}

// getLogGroupNames uses DescribeLogGroups API to retrieve LogGroupArn entries that matches the provided logGroupNamePrefix
func getLogGroupNames(svc *cloudwatchlogs.Client, logGroupNamePrefix string, withLinkedAccount bool) ([]string, error) {
	// construct DescribeLogGroupsInput
	describeLogGroupsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix:    awssdk.String(logGroupNamePrefix),
		IncludeLinkedAccounts: awssdk.Bool(withLinkedAccount),
	}

	// make API request
	var logGroupIDs []string
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, describeLogGroupsInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error DescribeLogGroups with Paginator: %w", err)
		}

		for _, lg := range page.LogGroups {
			logGroupIDs = append(logGroupIDs, *lg.LogGroupArn)
		}
	}
	return logGroupIDs, nil
}
