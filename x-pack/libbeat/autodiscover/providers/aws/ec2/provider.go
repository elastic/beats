// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	awsauto "github.com/elastic/beats/x-pack/libbeat/autodiscover/providers/aws"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
)

func init() {
	autodiscover.Registry.AddProvider("aws_ec2", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for aws EC2s.
type Provider struct {
	config        *awsauto.Config
	bus           bus.Bus
	templates     *template.Mapper
	startListener bus.Listener
	stopListener  bus.Listener
	watcher       *watcher
	uuid          uuid.UUID
}

// AutodiscoverBuilder is the main builder for this provider.
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Experimental("aws_ec2 autodiscover is experimental")

	config := awsauto.DefaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	awsCfg, err := awscommon.GetAWSCredentials(
		awscommon.ConfigAWS{
			AccessKeyID:     config.AWSConfig.AccessKeyID,
			SecretAccessKey: config.AWSConfig.SecretAccessKey,
			SessionToken:    config.AWSConfig.SessionToken,
			ProfileName:     config.AWSConfig.ProfileName,
		})

	// Construct MetricSet with a full regions list if there is no region specified.
	if config.Regions == nil {
		// set default region to make initial aws api call
		awsCfg.Region = "us-west-1"

		// check if endpoint is given from configuration
		if config.AWSConfig.Endpoint != "" {
			awsCfg.EndpointResolver = awssdk.ResolveWithEndpointURL("https://ec2." + awsCfg.Region + "." + config.AWSConfig.Endpoint)
		}
		svcEC2 := ec2.New(awsCfg)

		completeRegionsList, err := awsauto.GetRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		config.Regions = completeRegionsList
	}

	var clients []ec2iface.ClientAPI
	for _, region := range config.Regions {
		if err != nil {
			logp.Error(errors.Wrap(err, "error loading AWS config for aws_ec2 autodiscover provider"))
		}
		awsCfg.Region = region

		// check if endpoint is given from configuration
		if config.AWSConfig.Endpoint != "" {
			awsCfg.EndpointResolver = awssdk.ResolveWithEndpointURL("https://ec2." + region + "." + config.AWSConfig.Endpoint)
		}
		clients = append(clients, ec2.New(awsCfg))
	}

	return internalBuilder(uuid, bus, config, newAPIFetcher(clients))
}

// internalBuilder is mainly intended for testing via mocks and stubs.
// it can be configured to use a fetcher that doesn't actually hit the AWS API.
func internalBuilder(uuid uuid.UUID, bus bus.Bus, config *awsauto.Config, fetcher fetcher) (*Provider, error) {
	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		templates: &mapper,
		uuid:      uuid,
	}

	p.watcher = newWatcher(
		fetcher,
		config.Period,
		p.onWatcherStart,
		p.onWatcherStop,
	)

	return p, nil
}

// Start the autodiscover process.
func (p *Provider) Start() {
	p.watcher.start()
}

// Stop the autodiscover process.
func (p *Provider) Stop() {
	p.watcher.stop()
}

func (p *Provider) onWatcherStart(instanceID string, instance *ec2Instance) {
	e := bus.Event{
		"start":    true,
		"provider": p.uuid,
		"id":       instanceID,
		"aws": common.MapStr{
			"ec2": instance.toMap(),
		},
		"cloud": instance.toCloudMap(),
		"meta": common.MapStr{
			"aws": common.MapStr{
				"ec2": instance.toMap(),
			},
			"cloud": instance.toCloudMap(),
		},
	}

	if configs := p.templates.GetConfig(e); configs != nil {
		e["config"] = configs
	}
	p.bus.Publish(e)
}

func (p *Provider) onWatcherStop(instanceID string) {
	e := bus.Event{
		"stop":     true,
		"id":       instanceID,
		"provider": p.uuid,
	}
	p.bus.Publish(e)
}

func (p *Provider) String() string {
	return "aws_ec2"
}
