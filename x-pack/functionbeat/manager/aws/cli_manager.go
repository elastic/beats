// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	cf "github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/awslabs/goformation/v4/cloudformation/lambda"
	merrors "github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/core"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/executor"
	fnaws "github.com/elastic/beats/v7/x-pack/functionbeat/provider/aws/aws"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const (
	// AWS lambda currently support go 1.x as a runtime.
	runtime     = "go1.x"
	handlerName = "functionbeat-aws"
)

// AWSLambdaFunction add 'dependsOn' as a serializable parameters,  goformation doesn't currently
// serialize this field.
type AWSLambdaFunction struct {
	*lambda.Function
	DependsOn []string
}

type installer interface {
	Policies() []iam.Role_Policy
	Template() *cloudformation.Template
	LambdaConfig() *fnaws.LambdaConfig
}

// CLIManager interacts with the AWS Lambda API to deploy, update or remove a function.
// It will take care of creating the main lambda function and ask for each function type for the
// operation that need to be executed to connect the lambda to the triggers.
type CLIManager struct {
	templateBuilder *defaultTemplateBuilder
	awsCfg          aws.Config
	log             *logp.Logger
	config          *fnaws.Config
}

// stackName cloudformation stack are unique per function.
func (c *CLIManager) stackName(name string) string {
	return "fnb-" + name + "-stack"
}

func (c *CLIManager) deployTemplate(update bool, name string) error {
	templateData, err := c.templateBuilder.execute(name)
	if err != nil {
		return err
	}

	c.log.Debugf("Using cloudformation template:\n%s", templateData.json)

	_, err = c.awsCfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return fmt.Errorf("failed to retrieve aws credentials, please check AWS credential in config: %+v", err)
	}

	svcCF := cf.New(c.awsCfg)

	executer := executor.NewExecutor(c.log)
	executer.Add(newOpEnsureBucket(c.log, c.awsCfg, c.bucket()))
	executer.Add(newOpUploadToBucket(
		c.log,
		c.awsCfg,
		c.bucket(),
		templateData.codeKey,
		templateData.zip.content,
	))
	executer.Add(newOpUploadToBucket(
		c.log,
		c.awsCfg,
		c.bucket(),
		templateData.key,
		templateData.json,
	))
	if update {
		executer.Add(newOpUpdateCloudFormation(
			c.log,
			svcCF,
			templateData.url,
			c.stackName(name),
		))
	} else {
		executer.Add(newOpCreateCloudFormation(
			c.log,
			svcCF,
			templateData.url,
			c.stackName(name),
		))
	}

	executer.Add(newOpWaitCloudFormation(c.log, cf.New(c.awsCfg)))
	executer.Add(newOpDeleteFileBucket(c.log, c.awsCfg, c.bucket(), templateData.codeKey))

	ctx := newStackContext()
	if err := executer.Execute(ctx); err != nil {
		if rollbackErr := executer.Rollback(ctx); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// Deploy uploads the function to AWS.
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

	_, err := c.awsCfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return fmt.Errorf("failed to retrieve aws credentials, please check AWS credential in config: %+v", err)
	}

	svc := cf.New(c.awsCfg)
	executer := executor.NewExecutor(c.log)
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

// Export prints the exported function data.
func (c *CLIManager) Export(name string) error {
	tmpl, err := c.templateBuilder.RawTemplate(name)
	if err != nil {
		return err
	}

	fmt.Println(tmpl)

	return nil
}

// Package packages functions for AWS.
func (c *CLIManager) Package(outputPattern string) error {
	resource := zipResources()
	content, err := core.MakeZip(packageUncompressedLimit, packageCompressedLimit, resource)
	if err != nil {
		return err
	}

	output := strings.ReplaceAll(outputPattern, "{{.Provider}}", "aws")
	err = ioutil.WriteFile(output, content, 0644)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Generated package for provider aws at: %s\n", output)
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
	config := fnaws.DefaultConfig()
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	awsCfg, err := awscommon.InitializeAWSConfig(config.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credentials, please check AWS credential in config: %+v", err)
	}
	if config.Region != "" {
		awsCfg.Region = config.Region
	}

	builder, err := provider.TemplateBuilder()
	if err != nil {
		return nil, err
	}

	templateBuilder, ok := builder.(*defaultTemplateBuilder)
	if !ok {
		return nil, fmt.Errorf("not defaultTemplateBuilder")
	}

	return &CLIManager{
		config:          config,
		awsCfg:          awsCfg,
		log:             logp.NewLogger("aws"),
		templateBuilder: templateBuilder,
	}, nil
}
