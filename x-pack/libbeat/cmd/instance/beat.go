// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/instrumentation"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/plugin"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/otelmanager"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-ucfg"
)

// This is the timeout for the beat's internal publishing pipeline to close when shutting down the receiver. Closing
// requires flushing the event queue, and if this doesn't happen within the timeout, data may be lost depending on
// input type.
const receiverPublisherCloseTimeout = 5 * time.Second

// NewBeatForReceiver creates a Beat that will be used in the context of an otel receiver
func NewBeatForReceiver(settings instance.Settings, receiverConfig map[string]any, consumer consumer.Logs, componentID string, core zapcore.Core) (*instance.Beat, error) {
	b, err := instance.NewBeat(settings.Name,
		settings.IndexPrefix,
		settings.Version,
		settings.ElasticLicensed,
		settings.Initialize)
	if err != nil {
		return nil, err
	}

	b.Info.ComponentID = componentID
	b.Info.LogConsumer = consumer

	// begin code similar to configure
	if err = plugin.Initialize(); err != nil {
		return nil, fmt.Errorf("error initializing plugins: %w", err)
	}

	b.InputQueueSize = settings.InputQueueSize

	cfOpts := []ucfg.Option{
		ucfg.PathSep("."),
		ucfg.ResolveEnv,
		ucfg.VarExp,
	}

	err = setLogger(b, receiverConfig, core)
	if err != nil {
		return nil, fmt.Errorf("error configuring beats logger: %w", err)
	}

	// extracting it here for ease of use
	logger := b.Info.Logger

	// if output is set and if output is not otelconsumer, inform users
	if receiverConfig["output"] != nil && receiverConfig["output"].(map[string]any)["otelconsumer"] == nil { //nolint: errcheck // output will always be of map type
		logger.Debugf("configured output does not work with beatreceiver, please use appropriate exporter instead")
	}

	// all beatreceivers will use otelconsumer output by default
	receiverConfig["output"] = map[string]any{
		"otelconsumer": map[string]any{},
	}

	tmp, err := ucfg.NewFrom(receiverConfig, cfOpts...)
	if err != nil {
		return nil, fmt.Errorf("error converting receiver config to ucfg: %w", err)
	}

	cfg := (*config.C)(tmp)
	if settings.Name == "filebeat" {
		partialConfig := struct {
			Path paths.Path `config:"path"`
		}{}

		if err := cfg.Unpack(&partialConfig); err != nil {
			return nil, fmt.Errorf("error extracting default paths: %w", err)
		}
		p := paths.New()
		if err := p.InitPaths(&partialConfig.Path); err != nil {
			return nil, fmt.Errorf("error initializing default paths: %w", err)
		}
		b.Paths = p
	} else {
		if err := instance.InitPaths(cfg); err != nil {
			return nil, fmt.Errorf("error initializing paths: %w", err)
		}
		b.Paths = paths.Paths
	}

	// We have to initialize the keystore before any unpack or merging the cloud
	// options.
	store, err := instance.LoadKeystore(cfg, b.Info.Beat, b.Paths)
	if err != nil {
		return nil, fmt.Errorf("could not initialize the keystore: %w", err)
	}

	if settings.DisableConfigResolver {
		config.OverwriteConfigOpts([]ucfg.Option{
			ucfg.PathSep("."),
			ucfg.ResolveNOOP,
		})
	} else if store != nil {
		// TODO: Allow the options to be more flexible for dynamic changes
		// note that if the store is nil it should be excluded as an option
		config.OverwriteConfigOpts([]ucfg.Option{
			ucfg.PathSep("."),
			ucfg.Resolve(keystore.ResolverWrap(store)),
			ucfg.ResolveEnv,
			ucfg.VarExp,
		})
	}

	b.Monitoring = beat.NewMonitoring()

	b.SetKeystore(store)
	b.Beat.Keystore = store
	err = cloudid.OverwriteSettings(cfg)
	if err != nil {
		return nil, fmt.Errorf("error overwriting cloudid settings: %w", err)
	}

	b.RawConfig = cfg
	err = cfg.Unpack(&b.Config)
	if err != nil {
		return nil, fmt.Errorf("error unpacking config data: %w", err)
	}

	instrumentation, err := instrumentation.New(cfg, b.Info.Beat, b.Info.Version, logger)
	if err != nil {
		return nil, fmt.Errorf("error setting up instrumentation: %w", err)
	}
	b.Instrumentation = instrumentation

	if err := features.UpdateFromConfig(b.RawConfig); err != nil {
		return nil, fmt.Errorf("could not parse features: %w", err)
	}
	b.RegisterHostname(features.FQDN())

	b.Beat.Config = &b.Config.BeatConfig

	if name := b.Config.Name; name != "" {
		b.Info.Name = name
	}

	if err := common.SetTimestampPrecision(b.Config.TimestampPrecision); err != nil {
		return nil, fmt.Errorf("error setting timestamp precision: %w", err)
	}

	// log paths values to help with troubleshooting
	logger.Infof("%s", b.Paths.String())

	metaPath := b.Paths.Resolve(paths.Data, "meta.json")
	err = b.LoadMeta(metaPath)
	if err != nil {
		return nil, fmt.Errorf("error loading meta data: %w", err)
	}

	logger.Infof("Beat ID: %v", b.Info.ID)

	// Try to get the host's FQDN and set it.
	h, err := sysinfo.Host()
	if err != nil {
		return nil, fmt.Errorf("failed to get host information: %w", err)
	}

	fqdnLookupCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	fqdn, err := h.FQDNWithContext(fqdnLookupCtx)
	if err != nil {
		// FQDN lookup is "best effort".  We log the error, fallback to
		// the OS-reported hostname, and move on.
		logger.Warnf("unable to lookup FQDN: %s, using hostname = %s as FQDN", err.Error(), b.Info.Hostname)
		b.Info.FQDN = b.Info.Hostname
	} else {
		b.Info.FQDN = fqdn
	}

	// register NewOtelManager
	management.SetManagerFactory(otelmanager.NewOtelManager)

	// initialize config manager
	oCfg, _ := cfg.Child("management.otel", -1)
	m, err := management.NewManager(oCfg, b.Registry, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating new manager: %w", err)
	}
	b.Manager = m

	if b.Manager.AgentInfo().Version != "" {
		// During the manager initialization the client to connect to the agent is
		// also initialized. That makes the beat to read information sent by the
		// agent, which includes the AgentInfo with the agent's package version.
		// Components running under agent should report the agent's package version
		// as their own version.
		// In order to do so b.Info.Version needs to be set to the version the agent
		// sent. As this Beat instance is initialized much before the package
		// version is received, it's overridden here. So far it's early enough for
		// the whole beat to report the right version.
		b.Info.Version = b.Manager.AgentInfo().Version
		version.SetPackageVersion(b.Info.Version)
	}

	// build the user-agent string to be used by the outputs
	b.GenerateUserAgent()

	if err := b.Manager.CheckRawConfig(b.RawConfig); err != nil {
		return nil, fmt.Errorf("error checking raw config: %w", err)
	}

	b.Beat.BeatConfig, err = b.BeatConfig()
	if err != nil {
		return nil, fmt.Errorf("error setting BeatConfig: %w", err)
	}

	imFactory := settings.IndexManagement
	if imFactory == nil {
		imFactory = idxmgmt.MakeDefaultSupport(settings.ILM, logger)
	}
	b.IdxSupporter, err = imFactory(logger, b.Info, b.RawConfig)
	if err != nil {
		return nil, fmt.Errorf("error setting index supporter: %w", err)
	}

	processingFactory := settings.Processing
	if processingFactory == nil {
		processingFactory = processing.MakeDefaultBeatSupport(true)
	}

	processors, err := processingFactory(b.Info, logger.Named("processors"), b.RawConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating processors: %w", err)
	}
	b.SetProcessors(processors)

	reg := b.Monitoring.StatsRegistry().GetOrCreateRegistry("libbeat")

	monitors := pipeline.Monitors{
		Metrics:   reg,
		Telemetry: b.Monitoring.StateRegistry(),
		Logger:    logger.Named("publisher"),
		Tracer:    b.Instrumentation.Tracer(),
	}

	outputFactory := b.MakeOutputFactory(b.Config.Output)

	pipelineSettings := pipeline.Settings{
		Processors:     b.GetProcessors(),
		InputQueueSize: b.InputQueueSize,
		WaitCloseMode:  pipeline.WaitOnPipelineCloseThenForce,
		WaitClose:      receiverPublisherCloseTimeout,
	}
	publisher, err := pipeline.LoadWithSettings(b.Info, monitors, b.Config.Pipeline, outputFactory, pipelineSettings)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher: %w", err)
	}
	b.Registry.MustRegisterOutput(b.MakeOutputReloader(publisher.OutputReloader()))
	b.Publisher = publisher

	return b, nil
}

// setLogger configures a logp logger and sets it on b.Info.Logger
func setLogger(b *instance.Beat, receiverConfig map[string]any, core zapcore.Core) error {

	var err error
	logpConfig := logp.Config{}
	logpConfig.AddCaller = true
	logpConfig.Beat = b.Info.Beat
	logpConfig.Files.MaxSize = 1

	var logCfg *config.C
	if _, ok := receiverConfig["logging"]; !ok {
		logCfg = config.NewConfig()
	} else {
		logCfg = config.MustNewConfigFrom(receiverConfig["logging"])
	}

	if err := logCfg.Unpack(&logpConfig); err != nil {
		return fmt.Errorf("error unpacking beats logging config: %w\n%v", err, b.Config.Logging)
	}

	b.Info.Logger, err = logp.ConfigureWithCoreLocal(logpConfig, core)
	if err != nil {
		return fmt.Errorf("error configuring beats logp: %w", err)
	}

	return nil
}
