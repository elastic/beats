// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/awslabs/goformation/cloudformation"
	merrors "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
)

const (
	// AWS lambda currently support go 1.x as a runtime.
	runtime = "go1.x"

	bucket = "beatless-deploy"
)

type installer interface {
	Template() *cloudformation.Template
	LambdaConfig() *lambdaConfig
}

// CLIManager interacts with the AWS Lambda API to deploy, update or remove a function.
// It will take care of creating the main lambda function and ask for each function type for the
// operation that need to be executed to connect the lambda to the triggers.
type CLIManager struct {
	provider provider.Provider
	svc      *lambda.Lambda
	awsCfg   aws.Config
	log      *logp.Logger
}

func (c *CLIManager) findFunction(name string) (installer, error) {
	fn, err := c.provider.FindFunctionByName(name)
	if err != nil {
		return nil, err
	}

	function, ok := fn.(installer)
	if !ok {
		return nil, errors.New("incompatible type received, expecting: 'functionManager'")
	}

	return function, nil
}

func (c *CLIManager) template(function installer, name string) *cloudformation.Template {
	lambdaConfig := function.LambdaConfig()

	// Create the generate cloudformation template for the lambda itself.
	template := cloudformation.NewTemplate()
	template.Resources["IAMRoleLambdaExecution"] = &cloudformation.AWSIAMRole{
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": cloudformation.Join("", []string{
							"lambda.",
							cloudformation.Ref("AWS::URLSuffix"),
						}),
					},
				},
			},
		},
		RoleName: "beatless-lambda",
	}

	var dlc *cloudformation.AWSLambdaFunction_DeadLetterConfig
	if lambdaConfig.DeadLetterConfig != nil && len(lambdaConfig.DeadLetterConfig.TargetArn) != 0 {
		dlc = &cloudformation.AWSLambdaFunction_DeadLetterConfig{
			TargetArn: lambdaConfig.DeadLetterConfig.TargetArn,
		}
	}

	template.Resources["btl"+name] = &AWSLambdaFunction{
		AWSLambdaFunction: &cloudformation.AWSLambdaFunction{
			Code: &cloudformation.AWSLambdaFunction_Code{
				S3Bucket: bucket,
				S3Key:    c.codeKey(name),
			},
			Description: lambdaConfig.Description,
			Environment: &cloudformation.AWSLambdaFunction_Environment{
				Variables: map[string]string{
					"BEAT_STRICT_PERMS": "false",
					"ENABLED_FUNCTIONS": name,
				},
			},
			DeadLetterConfig:             dlc,
			FunctionName:                 name,
			Role:                         cloudformation.GetAtt("IAMRoleLambdaExecution", "Arn"),
			Runtime:                      runtime,
			Handler:                      handlerName,
			MemorySize:                   lambdaConfig.MemorySize.Megabytes(),
			ReservedConcurrentExecutions: lambdaConfig.Concurrency,
			Timeout: int(lambdaConfig.Timeout.Seconds()),
		},
		DependsOn: []string{"IAMRoleLambdaExecution"},
	}
	return template
}

// stackName cloudformation stack are unique per function.
func (c *CLIManager) stackName(name string) string {
	return "btl-" + name + "stack"
}

func (c *CLIManager) codeKey(name string) string {
	return "beatless-deployment/" + name + "/beatless.zip"
}

func (c *CLIManager) deployTemplate(update bool, name string) error {
	c.log.Debug("compressing assets")
	content, err := core.MakeZip()
	if err != nil {
		return err
	}
	c.log.Debugf("compression successful, zip size: %d bytes", len(content))

	function, err := c.findFunction(name)
	if err != nil {
		return err
	}

	fnTemplate := function.Template()

	template, err := mergeTemplate(c.template(function, name), fnTemplate)
	if err != nil {
		return err
	}

	json, err := template.JSON()
	if err != nil {
		return err
	}

	c.log.Debugf("cloudformation template: \n%s", json)

	context := &executorContext{}
	executer := newExecutor(c.log, context)
	executer.Add(newOpEnsureBucket(c.log, c.awsCfg, bucket))
	executer.Add(newOpUploadToBucket(c.log, c.awsCfg, bucket, c.codeKey(name), content))
	executer.Add(newOpUploadToBucket(
		c.log,
		c.awsCfg,
		bucket,
		"beatless-deployment/"+name+"/cloudformation-template-create.json",
		json,
	))
	if update {
		executer.Add(newOpUpdateCloudFormation(
			c.log,
			c.awsCfg,
			"https://s3.amazonaws.com/"+bucket+"/beatless-deployment/"+name+"/cloudformation-template-create.json",
			c.stackName(name),
		))
	} else {
		executer.Add(newOpCreateCloudFormation(
			c.log,
			c.awsCfg,
			"https://s3.amazonaws.com/"+bucket+"/beatless-deployment/"+name+"/cloudformation-template-create.json",
			c.stackName(name),
		))
	}

	executer.Add(newOpWaitCloudFormation(c.log, c.awsCfg, c.stackName(name)))

	if err := executer.Execute(); err != nil {
		if rollbackErr := executer.Rollback(); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// Deploy delegate deploy to the actual function implementation.
func (c *CLIManager) Deploy(name string) error {
	c.log.Debugf("function: %s, starting deploy", name)
	defer c.log.Debugf("function: %s, deploy execution ended", name)

	if err := c.deployTemplate(false, name); err != nil {
		return err
	}
	c.log.Debugf("Successfully created function: %s", name)
	return nil
}

// Update updates lambda using cloudformation.
func (c *CLIManager) Update(name string) error {
	c.log.Debugf("function: %s, starting update", name)
	defer c.log.Debugf("function: %s, update execution ended", name)

	if err := c.deployTemplate(true, name); err != nil {
		return err
	}

	c.log.Debugf("Successfully updated function: %s")
	return nil
}

// Remove removes a stack and unregister any resources created.
func (c *CLIManager) Remove(name string) error {
	c.log.Debugf("function: %s, starting remove", name)
	defer c.log.Debugf("function: %s, remove execution ended", name)

	context := &executorContext{}
	executer := newExecutor(c.log, context)
	executer.Add(newOpDeleteCloudFormation(c.log, c.awsCfg, c.stackName(name)))
	executer.Add(newWaitDeleteCloudFormation(c.log, c.awsCfg, c.stackName(name)))

	if err := executer.Execute(); err != nil {
		if rollbackErr := executer.Rollback(); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// NewCLI returns the interface to managa function on Amazon lambda.
func NewCLI(
	log *logp.Logger,
	cfg *common.Config,
	provider provider.Provider,
) (provider.CLIManager, error) {
	awsCfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}

	svc := lambda.New(awsCfg)
	return &CLIManager{
		provider: provider,
		svc:      svc,
		awsCfg:   awsCfg,
		log:      logp.NewLogger("aws lambda cli"),
	}, nil
}

// mergeTemplate takes two cloudformation and merge them, if a key already exist we return an error.
func mergeTemplate(one, two *cloudformation.Template) (*cloudformation.Template, error) {
	merge := func(m1 map[string]interface{}, m2 map[string]interface{}) (map[string]interface{}, error) {
		for k, v := range m2 {
			if _, ok := m1[k]; ok {
				return nil, fmt.Errorf("key %s already exist in the template map", k)
			}
			m1[k] = v
		}
		return m1, nil
	}

	v, err := merge(one.Parameters, two.Parameters)
	if err != nil {
		return nil, err
	}

	one.Parameters = v

	v, err = merge(one.Mappings, two.Mappings)
	if err != nil {
		return nil, err
	}
	one.Mappings = v

	v, err = merge(one.Conditions, two.Conditions)
	if err != nil {
		return nil, err
	}
	one.Conditions = v

	v, err = merge(one.Resources, two.Resources)
	if err != nil {
		return nil, err
	}
	one.Resources = v

	v, err = merge(one.Outputs, two.Outputs)
	if err != nil {
		return nil, err
	}

	one.Outputs = v
	return one, nil
}
