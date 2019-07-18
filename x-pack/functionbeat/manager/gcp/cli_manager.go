// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/functionbeat/manager/executor"
	fngcp "github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp"
)

type installer interface {
	Config() *fngcp.FunctionConfig
	Name() string
}

// CLIManager interacts with the AWS Lambda API to deploy, update or remove a function.
// It will take care of creating the main lambda function and ask for each function type for the
// operation that need to be executed to connect the lambda to the triggers.
type CLIManager struct {
	templateBuilder *defaultTemplateBuilder
	location        string
	tokenSrc        oauth2.TokenSource
	log             *logp.Logger
	config          *Config
}

// Deploy delegates deploy to the actual function implementation.
func (c *CLIManager) Deploy(name string) error {
	c.log.Debugf("Deploying function: %s", name)
	defer c.log.Debugf("Deploy finish for function '%s'", name)

	create := false
	err := c.deploy(create, name)
	if err != nil {
		return err
	}

	c.log.Debugf("Successfully created function '%s'", name)
	return nil
}

// Update updates the function.
func (c *CLIManager) Update(name string) error {
	c.log.Debugf("Starting updating function '%s'", name)
	defer c.log.Debugf("Update complete for function '%s'", name)

	update := true
	err := c.deploy(update, name)
	if err != nil {
		return err
	}

	c.log.Debugf("Successfully updated function: '%s'", name)
	return nil
}

// deploy uploads to bucket and creates a function on GCP.
func (c *CLIManager) deploy(update bool, name string) error {
	functionData, err := c.templateBuilder.execute(name)
	if err != nil {
		return err
	}

	executer := executor.NewExecutor(c.log)
	executer.Add(newOpEnsureBucket(c.log, c.config))
	executer.Add(newOpUploadToBucket(c.log, c.config, name, functionData.raw))

	if update {
		executer.Add(newOpUpdateFunction(c.log, c.tokenSrc, functionData.function.Name, functionData.function))
	} else {
		executer.Add(newOpCreateFunction(c.log, c.tokenSrc, c.location, name, functionData.function))
	}

	executer.Add(newOpWaitForFunction(c.log, c.tokenSrc))

	ctx := &functionContext{}
	if err := executer.Execute(ctx); err != nil {
		if rollbackErr := executer.Rollback(ctx); rollbackErr != nil {
			return errors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// Remove removes a stack and unregister any resources created.
func (c *CLIManager) Remove(name string) error {
	c.log.Debugf("Removing function: %s", name)
	defer c.log.Debugf("Removal of function '%s' complete", name)

	functionData, err := c.templateBuilder.execute(name)
	if err != nil {
		return err
	}

	executer := executor.NewExecutor(c.log)
	executer.Add(newOpDeleteFunction(c.log, c.location, functionData.function.Name, c.tokenSrc))
	executer.Add(newOpDeleteFromBucket(c.log, c.config, name))

	ctx := &functionContext{}
	if err := executer.Execute(ctx); err != nil {
		if rollbackErr := executer.Rollback(ctx); rollbackErr != nil {
			return errors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}

	c.log.Debugf("Successfully deleted function: '%s'", name)
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

// NewCLI returns the interface to manage function on Amazon lambda.
func NewCLI(
	log *logp.Logger,
	cfg *common.Config,
	provider provider.Provider,
) (provider.CLIManager, error) {
	config := &Config{}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}

	builder, err := provider.TemplateBuilder()
	if err != nil {
		return nil, err
	}

	templateBuilder, ok := builder.(*defaultTemplateBuilder)
	if !ok {
		return nil, fmt.Errorf("not defaultTemplateBuilder")
	}

	location := fmt.Sprintf(locationTemplate, config.ProjectID, config.Location)

	tokenSrc, err := google.DefaultTokenSource(context.TODO(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("error while creating CLIManager: %+v", err)
	}

	return &CLIManager{
		config:          config,
		log:             logp.NewLogger("gcp"),
		location:        location,
		tokenSrc:        tokenSrc,
		templateBuilder: templateBuilder,
	}, nil
}
