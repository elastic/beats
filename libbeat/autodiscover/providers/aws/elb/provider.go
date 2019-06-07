package elb

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/gofrs/uuid"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

func init() {
	autodiscover.Registry.AddProvider("aws_elb", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for aws ELBs.
type Provider struct {
	fetcher       fetcher
	config        *Config
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
	cfgwarn.Beta("aws_elb autodiscover is beta")

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	awsCfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		logp.Err("error loading AWS config for aws_elb autodiscover provider: %s", err)
	}

	awsCreds := config.Credentials
	if awsCreds != nil && awsCreds.AccessKeyID != "" && awsCreds.SecretAccessKey != "" {
		awsCfg.Credentials = aws.NewStaticCredentialsProvider(awsCreds.AccessKeyID, awsCreds.SecretAccessKey, "")
	}
	awsCfg.Region = config.Region

	return internalBuilder(uuid, bus, config, newAPIFetcher(elasticloadbalancingv2.New(awsCfg)))
}

// internalBuilder is mainly intended for testing via mocks and stubs.
// it can be configured to use a fetcher that doesn't actually hit the AWS API.
func internalBuilder(uuid uuid.UUID, bus bus.Bus, config *Config, fetcher fetcher) (*Provider, error) {
	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, nil)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	return &Provider{
		fetcher:   fetcher,
		config:    config,
		bus:       bus,
		builders:  builders,
		appenders: appenders,
		templates: &mapper,
		uuid:      uuid,
	}, nil
}

// Start the autodiscover process.
func (p *Provider) Start() {
	p.watcher = newWatcher(
		p.fetcher,
		10*time.Second,
		p.onWatcherStart,
		p.onWatcherStop,
	)
	p.watcher.start()
}

// Stop the autodiscover process.
func (p *Provider) Stop() {
	p.watcher.stop()
}

func (p *Provider) onWatcherStart(arn string, lbl *lbListener) {
	lblMap := lbl.toMap()
	e := bus.Event{
		"start":        true,
		"provider":     p.uuid,
		"id":           arn,
		"host":         lblMap["host"],
		"port":         lblMap["port"],
		"elb_listener": lbl.toMap(),
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
