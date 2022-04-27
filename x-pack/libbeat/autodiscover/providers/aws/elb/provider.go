// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/elasticloadbalancingv2iface"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
	awsauto "github.com/elastic/beats/v7/x-pack/libbeat/autodiscover/providers/aws"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
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
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *common.Config,
	keystore keystore.Keystore,
) (autodiscover.Provider, error) {
	cfgwarn.Experimental("aws_elb autodiscover is experimental")

	config := awsauto.DefaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	awsCfg, err := awscommon.InitializeAWSConfig(awscommon.ConfigAWS{
		AccessKeyID:     config.AWSConfig.AccessKeyID,
		SecretAccessKey: config.AWSConfig.SecretAccessKey,
		SessionToken:    config.AWSConfig.SessionToken,
		ProfileName:     config.AWSConfig.ProfileName,
	})

	// Construct MetricSet with a full regions list if there is no region specified.
	if config.Regions == nil {
		ec2ServiceName := awscommon.CreateServiceName("ec2", config.AWSConfig.FIPSEnabled, awsCfg.Region)
		svcEC2 := ec2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, ec2ServiceName, awsCfg.Region, awsCfg))

		completeRegionsList, err := awsauto.GetRegions(svcEC2)
		if err != nil {
			return nil, err
		}

		config.Regions = completeRegionsList
	}

	var clients []elasticloadbalancingv2iface.ClientAPI
	for _, region := range config.Regions {
		awsCfg, err := awscommon.InitializeAWSConfig(awscommon.ConfigAWS{
			AccessKeyID:     config.AWSConfig.AccessKeyID,
			SecretAccessKey: config.AWSConfig.SecretAccessKey,
			SessionToken:    config.AWSConfig.SessionToken,
			ProfileName:     config.AWSConfig.ProfileName,
		})
		if err != nil {
			logp.Err("error loading AWS config for aws_elb autodiscover provider: %s", err)
		}
		awsCfg.Region = region
		elbServiceName := awscommon.CreateServiceName("elasticloadbalancing", config.AWSConfig.FIPSEnabled, region)
		clients = append(clients, elasticloadbalancingv2.New(awscommon.EnrichAWSConfigWithEndpoint(
			config.AWSConfig.Endpoint, elbServiceName, region, awsCfg)))
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

func (p *Provider) onWatcherStart(arn string, lbl *lbListener) {
	lblMap := lbl.toMap()
	e := bus.Event{
		"start":    true,
		"provider": p.uuid,
		"id":       arn,
		"host":     lblMap["host"],
		"port":     lblMap["port"],
		"aws": mapstr.M{
			"elb": lbl.toMap(),
		},
		"cloud": lbl.toCloudMap(),
		"meta": mapstr.M{
			"aws": mapstr.M{
				"elb": lbl.toMap(),
			},
			"cloud": lbl.toCloudMap(),
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
