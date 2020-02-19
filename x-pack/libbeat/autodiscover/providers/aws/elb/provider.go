// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/elasticloadbalancingv2iface"
	"github.com/gofrs/uuid"

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
	autodiscover.Registry.AddProvider("aws_elb", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for aws ELBs.
type Provider struct {
	config        *awsauto.Config
	bus           bus.Bus
	builders      autodiscover.Builders
	appenders     autodiscover.Appenders
	templates     *template.Mapper
	startListener bus.Listener
	stopListener  bus.Listener
	watcher       *watcher
	uuid          uuid.UUID
}

// AutodiscoverBuilder is the main builder for this provider.
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Experimental("aws_elb autodiscover is experimental")

	config := awsauto.DefaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	awsCfg, err := awscommon.GetAWSCredentials(awscommon.ConfigAWS{
		AccessKeyID:     config.AWSConfig.AccessKeyID,
		SecretAccessKey: config.AWSConfig.SecretAccessKey,
		SessionToken:    config.AWSConfig.SessionToken,
		ProfileName:     config.AWSConfig.ProfileName,
	})

	// Construct MetricSet with a full regions list if there is no region specified.
	if config.Regions == nil {
		// set default region to make initial aws api call
		awsCfg.Region = "us-west-1"
		svcEC2 := ec2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, "ec2", awsCfg.Region, awsCfg))

		completeRegionsList, err := awsauto.GetRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		config.Regions = completeRegionsList
	}

	var clients []elasticloadbalancingv2iface.ClientAPI
	for _, region := range config.Regions {
		awsCfg, err := awscommon.GetAWSCredentials(awscommon.ConfigAWS{
			AccessKeyID:     config.AWSConfig.AccessKeyID,
			SecretAccessKey: config.AWSConfig.SecretAccessKey,
			SessionToken:    config.AWSConfig.SessionToken,
			ProfileName:     config.AWSConfig.ProfileName,
		})
		if err != nil {
			logp.Err("error loading AWS config for aws_elb autodiscover provider: %s", err)
		}
		awsCfg.Region = region
		clients = append(clients, elasticloadbalancingv2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, "elasticloadbalancing", region, awsCfg)))
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

func (p *Provider) onWatcherStart(arn string, lbl *lbListener) {
	lblMap := lbl.toMap()
	e := bus.Event{
		"start":    true,
		"provider": p.uuid,
		"id":       arn,
		"host":     lblMap["host"],
		"port":     lblMap["port"],
		"meta": common.MapStr{
			"elb_listener": lbl.toMap(),
			"cloud":        lbl.toCloudMap(),
		},
	}

	if configs := p.templates.GetConfig(e); configs != nil {
		e["config"] = configs
	}
	p.appenders.Append(e)
	p.bus.Publish(e)
}

func (p *Provider) onWatcherStop(arn string) {
	e := bus.Event{
		"stop":     true,
		"id":       arn,
		"provider": p.uuid,
	}
	p.bus.Publish(e)
}

func (p *Provider) String() string {
	return "aws_elb"
}
