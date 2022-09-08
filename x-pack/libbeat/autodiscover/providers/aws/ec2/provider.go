// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	_ = autodiscover.Registry.AddProvider("aws_ec2", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for aws EC2s.
type Provider struct {
	config    *awsauto.Config
	bus       bus.Bus
	templates *template.Mapper
	watcher   *watcher
	uuid      uuid.UUID
}

// AutodiscoverBuilder is the main builder for this provider.
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *conf.C,
	keystore keystore.Keystore,
) (autodiscover.Provider, error) {
	cfgwarn.Experimental("aws_ec2 autodiscover is experimental")

	config := awsauto.DefaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	awsCfg, err := awscommon.InitializeAWSConfig(
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
		svcEC2 := ec2.NewFromConfig(awsCfg, func(o *ec2.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		})

		completeRegionsList, err := awsauto.GetRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		config.Regions = completeRegionsList
	}

	var clients []ec2.DescribeInstancesAPIClient
	for _, region := range config.Regions {
		if err != nil {
			logp.Error(fmt.Errorf("error loading AWS config for aws_ec2 autodiscover provider: %w", err))
		}
		awsCfg.Region = region
		clients = append(clients, ec2.NewFromConfig(awsCfg, func(o *ec2.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		}))
	}

	return internalBuilder(uuid, bus, config, newAPIFetcher(clients), keystore)
}

// internalBuilder is mainly intended for testing via mocks and stubs.
// it can be configured to use a fetcher that doesn't actually hit the AWS API.
func internalBuilder(uuid uuid.UUID, bus bus.Bus, config *awsauto.Config, fetcher fetcher, keystore keystore.Keystore) (*Provider, error) {
	mapper, err := template.NewConfigMapper(config.Templates, keystore, nil)
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
		"aws": mapstr.M{
			"ec2": instance.toMap(),
		},
		"cloud": instance.toCloudMap(),
		"meta": mapstr.M{
			"aws": mapstr.M{
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
