// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	cf "github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/awslabs/goformation/cloudformation"
	merrors "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/core"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
)

const (
	// AWS lambda currently support go 1.x as a runtime.
	runtime     = "go1.x"
	handlerName = "functionbeat"

	// invalidChars for resource name
	invalidChars = ":-/"
)

// AWSLambdaFunction add 'dependsOn' as a serializable parameters, for no good reason it's
// not supported.
type AWSLambdaFunction struct {
	*cloudformation.AWSLambdaFunction
	DependsOn []string
}

type installer interface {
	Template() *cloudformation.Template
	LambdaConfig() *lambdaConfig
}

// CLIManager interacts with the AWS Lambda API to deploy, update or remove a function.
// It will take care of creating the main lambda function and ask for each function type for the
// operation that need to be executed to connect the lambda to the triggers.
type CLIManager struct {
	provider provider.Provider
	awsCfg   aws.Config
	log      *logp.Logger
	config   *Config
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

func (c *CLIManager) template(function installer, name, codeLoc string) *cloudformation.Template {
	lambdaConfig := function.LambdaConfig()

	prefix := func(s string) string {
		return "fnb" + name + s
	}

	// AWS variables references:.
	// AWS::Partition: aws, aws-cn, aws-gov.
	// AWS::Region: us-east-1, us-east-2, ap-northeast-3,
	// AWS::AccountId: account id for the current request.
	// AWS::URLSuffix: amazonaws.com
	//
	// Documentation: https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/Welcome.html
	// Intrinsic function reference: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html

	// Create the roles for the lambda.
	template := cloudformation.NewTemplate()
	// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-role.html
	template.Resources["IAMRoleLambdaExecution"] = &cloudformation.AWSIAMRole{
		AssumeRolePolicyDocument: map[string]interface{}{
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
		Path:     "/",
		RoleName: "functionbeat-lambda-" + name,
		// Allow the lambda to write log to cloudwatch logs.
		// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-policy.html
		Policies: []cloudformation.AWSIAMRole_Policy{
			cloudformation.AWSIAMRole_Policy{
				PolicyName: cloudformation.Join("-", []string{"fnb", "lambda", name}),
				PolicyDocument: map[string]interface{}{
					"Statement": []map[string]interface{}{
						map[string]interface{}{
							"Action": []string{"logs:CreateLogStream", "Logs:PutLogEvents"},
							"Effect": "Allow",
							"Resource": []string{
								cloudformation.Sub("arn:${AWS::Partition}:logs:${AWS::Region}:${AWS::AccountId}:log-group:/aws/lambda/" + name + ":*"),
							},
						},
					},
				},
			},
		},
	}

	// Configure the Dead letter, any failed events will be send to the configured amazon resource name.
	var dlc *cloudformation.AWSLambdaFunction_DeadLetterConfig
	if lambdaConfig.DeadLetterConfig != nil && len(lambdaConfig.DeadLetterConfig.TargetArn) != 0 {
		dlc = &cloudformation.AWSLambdaFunction_DeadLetterConfig{
			TargetArn: lambdaConfig.DeadLetterConfig.TargetArn,
		}
	}

	// Create the lambda
	// Doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html
	template.Resources[prefix("")] = &AWSLambdaFunction{
		AWSLambdaFunction: &cloudformation.AWSLambdaFunction{
			Code: &cloudformation.AWSLambdaFunction_Code{
				S3Bucket: c.bucket(),
				S3Key:    codeLoc,
			},
			Description: lambdaConfig.Description,
			Environment: &cloudformation.AWSLambdaFunction_Environment{
				// Configure which function need to be run by the lambda function.
				Variables: map[string]string{
					"BEAT_STRICT_PERMS": "false", // Disable any check on disk, we are running with really differents permission on lambda.
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

	// Create the log group for the specific function lambda.
	template.Resources[prefix("LogGroup")] = &cloudformation.AWSLogsLogGroup{
		LogGroupName: "/aws/lambda/" + name,
	}

	return template
}

// stackName cloudformation stack are unique per function.
func (c *CLIManager) stackName(name string) string {
	return "fnb-" + name + "-stack"
}

func (c *CLIManager) deployTemplate(update bool, name string) error {
	c.log.Debug("Compressing all assets into an artifact")
	content, err := core.MakeZip()
	if err != nil {
		return err
	}
	c.log.Debugf("Compression is successful (zip size: %d bytes)", len(content))

	function, err := c.findFunction(name)
	if err != nil {
		return err
	}

	fnTemplate := function.Template()

	zipChecksum := checksum(content)
	codeKey := "functionbeat-deployment/" + name + "/" + zipChecksum + "/functionbeat.zip"

	to := c.template(function, name, codeKey)
	if err := mergeTemplate(to, fnTemplate); err != nil {
		return err
	}

	json, err := to.JSON()
	if err != nil {
		return err
	}

	templateChecksum := checksum(json)
	templateKey := "functionbeat-deployment/" + name + "/" + templateChecksum + "/cloudformation-template-create.json"
	templateURL := "https://s3.amazonaws.com/" + c.bucket() + "/" + templateKey

	c.log.Debugf("Using cloudformation template:\n%s", json)
	svcCF := cf.New(c.awsCfg)

	executer := newExecutor(c.log)
	executer.Add(newOpEnsureBucket(c.log, c.awsCfg, c.bucket()))
	executer.Add(newOpUploadToBucket(
		c.log,
		c.awsCfg,
		c.bucket(),
		codeKey,
		content,
	))
	executer.Add(newOpUploadToBucket(
		c.log,
		c.awsCfg,
		c.bucket(),
		templateKey,
		json,
	))
	if update {
		executer.Add(newOpUpdateCloudFormation(
			c.log,
			svcCF,
			templateURL,
			c.stackName(name),
		))
	} else {
		executer.Add(newOpCreateCloudFormation(
			c.log,
			svcCF,
			templateURL,
			c.stackName(name),
		))
	}

	executer.Add(newOpWaitCloudFormation(c.log, cf.New(c.awsCfg)))
	executer.Add(newOpDeleteFileBucket(c.log, c.awsCfg, c.bucket(), codeKey))

	ctx := newStackContext()
	if err := executer.Execute(ctx); err != nil {
		if rollbackErr := executer.Rollback(ctx); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// Deploy delegate deploy to the actual function implementation.
func (c *CLIManager) Deploy(name string) error {
	c.log.Debugf("Deploying function: %s", name)
	defer c.log.Debugf("Deploy finish for function '%s'", name)

	if err := c.deployTemplate(false, name); err != nil {
		return err
	}
	c.log.Debugf("Successfully created function '%s'", name)
	return nil
}

// Update updates lambda using cloudformation.
func (c *CLIManager) Update(name string) error {
	c.log.Debugf("Starting updating function '%s'", name)
	defer c.log.Debugf("Update complete for function '%s'", name)

	if err := c.deployTemplate(true, name); err != nil {
		return err
	}

	c.log.Debugf("Successfully updated function: '%s'", name)
	return nil
}

// Remove removes a stack and unregister any resources created.
func (c *CLIManager) Remove(name string) error {
	c.log.Debugf("Removing function: %s", name)
	defer c.log.Debugf("Removal of function '%s' complete", name)

	svc := cf.New(c.awsCfg)
	executer := newExecutor(c.log)
	executer.Add(newOpDeleteCloudFormation(c.log, svc, c.stackName(name)))
	executer.Add(newWaitDeleteCloudFormation(c.log, c.awsCfg))

	ctx := newStackContext()
	if err := executer.Execute(ctx); err != nil {
		if rollbackErr := executer.Rollback(ctx); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

func (c *CLIManager) bucket() string {
	return string(c.config.DeployBucket)
}

// NewCLI returns the interface to manage function on Amazon lambda.
func NewCLI(
	log *logp.Logger,
	cfg *common.Config,
	provider provider.Provider,
) (provider.CLIManager, error) {
	awsCfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}

	return &CLIManager{
		config:   config,
		provider: provider,
		awsCfg:   awsCfg,
		log:      logp.NewLogger("aws"),
	}, nil
}

// mergeTemplate takes two cloudformation and merge them, if a key already exist we return an error.
func mergeTemplate(to, from *cloudformation.Template) error {
	merge := func(m1 map[string]interface{}, m2 map[string]interface{}) error {
		for k, v := range m2 {
			if _, ok := m1[k]; ok {
				return fmt.Errorf("key %s already exist in the template map", k)
			}
			m1[k] = v
		}
		return nil
	}

	err := merge(to.Parameters, from.Parameters)
	if err != nil {
		return err
	}

	err = merge(to.Mappings, from.Mappings)
	if err != nil {
		return err
	}

	err = merge(to.Conditions, from.Conditions)
	if err != nil {
		return err
	}

	err = merge(to.Resources, from.Resources)
	if err != nil {
		return err
	}

	err = merge(to.Outputs, from.Outputs)
	if err != nil {
		return err
	}

	return nil
}

func normalizeResourceName(s string) string {
	return common.RemoveChars(s, invalidChars)
}

func checksum(data []byte) string {
	sha := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(sha[:])
}
