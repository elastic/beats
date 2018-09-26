// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	bc "github.com/elastic/beats/x-pack/beatless/config"
	"github.com/elastic/beats/x-pack/beatless/core/bundle"
	"github.com/elastic/beats/x-pack/beatless/provider"
)

const (
	// AWS lambda currently support go 1.x as a runtime.
	runtime = "go1.x"

	// Package size limits for AWS lambda, we should be a lot under this limit but
	// adding a check to make sure we never go over.
	packageCompressedLimit   = 50 * 1000 * 1000  // 50MB
	packageUncompressedLimit = 250 * 1000 * 1000 // 250MB

	handlerName = "beatless"
)

type functionManager interface {
	Deploy([]byte, aws.Config) error
	Update([]byte, aws.Config) error
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

func (c *CLIManager) rawYaml() ([]byte, error) {
	// Load the configuration file from disk with all the settings,
	// the function takes care of using -c.
	rawConfig, err := cfgfile.Load("", bc.ConfigOverrides)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, err
	}

	res, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *CLIManager) makeZip() ([]byte, error) {
	rawConfig, err := c.rawYaml()
	if err != nil {
		return nil, err
	}
	bundle := bundle.NewZipWithLimits(
		packageUncompressedLimit,
		packageCompressedLimit,
		&bundle.MemoryFile{Path: "beatless.yml", Raw: rawConfig, FileMode: 0766},
		&bundle.LocalFile{Path: "build/beatless", FileMode: 0755},
	)

	c.log.Debug("compressing assets")
	content, err := bundle.Bytes()
	if err != nil {
		return nil, err
	}
	c.log.Debugf("compression successful, zip size: %d bytes", len(content))
	return content, nil
}

func (c *CLIManager) findFunction(name string) (functionManager, error) {
	fn, err := c.provider.FindFunctionByName(name)
	if err != nil {
		return nil, err
	}

	function, ok := fn.(functionManager)
	if !ok {
		return nil, errors.New("incompatible type received, expecting: 'functionManager'")
	}

	return function, nil
}

// Deploy delegate deploy to the actual function implementation.
func (c *CLIManager) Deploy(name string) error {
	c.log.Debugf("function: %s, starting deploy", name)
	defer c.log.Debugf("function: %s, deploy execution ended", name)

	content, err := c.makeZip()
	if err != nil {
		return err
	}

	function, err := c.findFunction(name)
	if err != nil {
		return err
	}

	if err := function.Deploy(content, c.awsCfg); err != nil {
		return err
	}

	c.log.Debugf("Successfully created function: %s", name)
	return nil
}

// TODO add support for version qualifier
func (c *CLIManager) Update(name string) error {
	c.log.Debugf("function: %s, starting update", name)
	defer c.log.Debugf("function: %s, update execution ended", name)

	content, err := c.makeZip()
	if err != nil {
		return err
	}

	function, err := c.findFunction(name)
	if err != nil {
		return err
	}

	if err := function.Update(content, c.awsCfg); err != nil {
		return err
	}

	c.log.Debugf("Successfully updated function: %s")
	return nil
}

// TODO add support for version qualifier
// TODO add support for force to remove function not in the YML?
func (c *CLIManager) Remove(name string) error {
	c.log.Debugf("function: %s, starting remove", name)
	defer c.log.Debugf("function: %s, remove execution ended", name)

	deleteReq := &lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	}
	req := c.svc.DeleteFunctionRequest(deleteReq)

	resp, err := req.Send()
	if err != nil {
		c.log.Debugf("could not remove function: %s, error: %s, response:", name, err, resp)
		return err
	}

	c.log.Debugf("Removal successful of function: %s, response: %s", name, resp)
	return nil
}

func NewCLI(
	log *logp.Logger,
	cfg *common.Config,
	provider provider.Provider,
) (provider.CLIManager, error) {
	// TODO use configuration from the yml file
	// correctly merge with priority.
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
