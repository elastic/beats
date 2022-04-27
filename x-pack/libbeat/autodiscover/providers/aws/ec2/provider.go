// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
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
		ec2ServiceName := awscommon.CreateServiceName("ec2", config.AWSConfig.FIPSEnabled, awsCfg.Region)
		svcEC2 := ec2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, ec2ServiceName, awsCfg.Region, awsCfg))

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
		ec2ServiceName := awscommon.CreateServiceName("ec2", config.AWSConfig.FIPSEnabled, region)
		clients = append(clients, ec2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, ec2ServiceName, region, awsCfg)))
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
