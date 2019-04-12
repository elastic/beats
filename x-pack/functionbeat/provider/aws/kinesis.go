// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/goformation/cloudformation"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/core"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/aws/transformer"
)

type startingPosition uint

const (
	// Looking at the documentation, Kinesis should also support `AT_TIMESTAMP` but looking at the
	// request format for cloudformation, I don't see a way to define the timestamp.
	// I've looked at other frameworks, and it seems a bug in the cloudformation API.
	// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-eventsourcemapping.html
	trimHorizonPos startingPosition = iota + 1
	latestPos
)

var (
	mapStartingPosition = map[string]startingPosition{
		"trim_horizon": trimHorizonPos,
		"latest":       latestPos,
	}

	mapStartingPositionReverse = make(map[startingPosition]string, len(mapStartingPosition))
)

func init() {
	for k, v := range mapStartingPosition {
		mapStartingPositionReverse[v] = strings.ToUpper(k)
	}
}

func (s *startingPosition) Unpack(str string) error {
	v, ok := mapStartingPosition[str]
	if !ok {
		validValues := make([]string, len(mapStartingPosition))
		pos := 0
		for k := range mapStartingPosition {
			validValues[pos] = k
			pos++
		}
		return fmt.Errorf("unknown value %s, valid values are: %s", str, strings.Join(validValues, ", "))
	}
	*s = v
	return nil
}

func (s *startingPosition) String() string {
	v, ok := mapStartingPositionReverse[*s]
	if !ok {
		panic("unknown starting position: " + string(*s))
	}
	return v
}

// KinesisConfig is the configuration for the Kinesis event type.
type KinesisConfig struct {
	Description  string                  `config:"description"`
	Name         string                  `config:"name" validate:"nonzero,required"`
	Triggers     []*KinesisTriggerConfig `config:"triggers"`
	LambdaConfig *lambdaConfig           `config:",inline"`
}

// Validate validates the configuration.
func (cfg *KinesisConfig) Validate() error {
	if len(cfg.Triggers) == 0 {
		return errors.New("you need to specify at least one trigger")
	}
	return nil
}

// KinesisTriggerConfig configuration for the current trigger.
type KinesisTriggerConfig struct {
	EventSourceArn   string           `config:"event_source_arn" validate:"required"`
	BatchSize        int              `config:"batch_size" validate:"min=100,max=10000"`
	StartingPosition startingPosition `config:"starting_position"`
}

// Unpack unpacks the trigger and make sure the defaults settings are correctly sets.
func (c *KinesisTriggerConfig) Unpack(cfg *common.Config) error {
	type tmpConfig KinesisTriggerConfig
	config := tmpConfig{
		BatchSize:        100,
		StartingPosition: trimHorizonPos,
	}
	if err := cfg.Unpack(&config); err != nil {
		return err
	}
	*c = KinesisTriggerConfig(config)
	return nil
}

// Kinesis receives events from a kinesis stream and forward them to elasticsearch.
type Kinesis struct {
	log    *logp.Logger
	config *KinesisConfig
}

// NewKinesis creates a new function to receives events from a kinesis stream.
func NewKinesis(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := &KinesisConfig{LambdaConfig: DefaultLambdaConfig}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &Kinesis{log: logp.NewLogger("kinesis"), config: config}, nil
}

// Run starts the lambda function and wait for web triggers.
func (k *Kinesis) Run(_ context.Context, client core.Client) error {
	lambda.Start(k.createHandler(client))
	return nil
}

func (k *Kinesis) createHandler(client core.Client) func(request events.KinesisEvent) error {
	return func(request events.KinesisEvent) error {
		k.log.Debugf("The handler receives %d events", len(request.Records))

		events := transformer.KinesisEvent(request)
		if err := client.PublishAll(events); err != nil {
			k.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (k *Kinesis) Name() string {
	return "kinesis"
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (k *Kinesis) LambdaConfig() *lambdaConfig {
	return k.config.LambdaConfig
}

// Template returns the cloudformation template for configuring the service with the specified
// triggers.
func (k *Kinesis) Template() *cloudformation.Template {
	template := cloudformation.NewTemplate()
	prefix := func(suffix string) string {
		return normalizeResourceName("fnb" + k.config.Name + k.Name() + suffix)
	}

	for _, trigger := range k.config.Triggers {
		resourceName := prefix(trigger.EventSourceArn)
		template.Resources[resourceName] = &cloudformation.AWSLambdaEventSourceMapping{
			BatchSize:        trigger.BatchSize,
			EventSourceArn:   trigger.EventSourceArn,
			FunctionName:     cloudformation.GetAtt("fnb"+k.config.Name, "Arn"),
			StartingPosition: trigger.StartingPosition.String(),
		}
	}

	return template
}

// Policies returns a slice of policy to add to the lambda role.
func (k *Kinesis) Policies() []cloudformation.AWSIAMRole_Policy {
	resources := make([]string, len(k.config.Triggers))
	for idx, trigger := range k.config.Triggers {
		resources[idx] = trigger.EventSourceArn
	}

	// Give us a chance to generate the same document indenpendant of the changes,
	// to help with updates.
	sort.Strings(resources)

	policies := []cloudformation.AWSIAMRole_Policy{
		cloudformation.AWSIAMRole_Policy{
			PolicyName: cloudformation.Join("-", []string{"fnb", "kinesis", k.config.Name}),
			PolicyDocument: map[string]interface{}{
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": []string{
							"kinesis:GetRecords",
							"kinesis:GetShardIterator",
							"Kinesis:DescribeStream",
						},
						"Effect":   "Allow",
						"Resource": resources,
					},
				},
			},
		},
	}

	return policies
}
