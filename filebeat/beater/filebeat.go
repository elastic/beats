// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/filebeat/channel"
	cfg "github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/fileset"
	_ "github.com/elastic/beats/v7/filebeat/include"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/filestream"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/compat"
	"github.com/elastic/beats/v7/filebeat/registrar"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/go-concert/unison"

	// Add filebeat level processors
	_ "github.com/elastic/beats/v7/filebeat/processor/add_kubernetes_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/decode_csv_fields"

	// include all filebeat specific autodiscover features
	_ "github.com/elastic/beats/v7/filebeat/autodiscover"
)

const pipelinesWarning = "Filebeat is unable to load the ingest pipelines for the configured" +
	" modules because the Elasticsearch output is not configured/enabled. If you have" +
	" already loaded the ingest pipelines or are using Logstash pipelines, you" +
	" can ignore this warning."

var once = flag.Bool("once", false, "Run filebeat only once until all harvesters reach EOF")

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config                   *cfg.Config
	moduleRegistry           *fileset.ModuleRegistry
	pluginFactory            PluginFactory
	done                     chan struct{}
	stopOnce                 sync.Once // wraps the Stop() method
	pipeline                 beat.PipelineConnector
	logger                   *logp.Logger
	otelStatusFactoryWrapper func(cfgfile.RunnerFactory) cfgfile.RunnerFactory
}

type PluginFactory func(beat.Info, *logp.Logger, statestore.States, *paths.Path) []v2.Plugin

// New creates a new Filebeat pointer instance.
func New(plugins PluginFactory) beat.Creator {
	return func(b *beat.Beat, rawConfig *conf.C) (beat.Beater, error) {
		return newBeater(b, plugins, rawConfig)
	}
}

func newBeater(b *beat.Beat, plugins PluginFactory, rawConfig *conf.C) (beat.Beater, error) {
	config := cfg.DefaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %w", err) //nolint:staticcheck //Keep old behavior
	}

	if err := cfgwarn.CheckRemoved6xSettings(
		rawConfig,
		"prospectors",
		"config.prospectors",
		"config_dir",
		"registry_file",
		"registry_file_permissions",
		"registry_flush",
	); err != nil {
		return nil, err
	}

	enableAllFilesets, _ := b.BeatConfig.Bool("config.modules.enable_all_filesets", -1)
	forceEnableModuleFilesets, _ := b.BeatConfig.Bool("config.modules.force_enable_module_filesets", -1)
	filesetOverrides := fileset.FilesetOverrides{
		EnableAllFilesets:         enableAllFilesets,
		ForceEnableModuleFilesets: forceEnableModuleFilesets,
	}
	moduleRegistry, err := fileset.NewModuleRegistry(config.Modules, b.Info, true, filesetOverrides, b.Paths)
	if err != nil {
		return nil, err
	}

	moduleInputs, err := moduleRegistry.GetInputConfigs()
	if err != nil {
		return nil, err
	}

	if b.API != nil {
		if err = inputmon.AttachHandler(b.API.Router(), b.Monitoring.InputsRegistry()); err != nil {
			return nil, fmt.Errorf("failed attach inputs api to monitoring endpoint server: %w", err)
		}
	}

	// Add inputs created by the modules
	config.Inputs = append(config.Inputs, moduleInputs...)

	enabledInputs := config.ListEnabledInputs()
	var haveEnabledInputs bool
	if len(enabledInputs) > 0 {
		haveEnabledInputs = true
	}

	if !config.ConfigInput.Enabled() && !config.ConfigModules.Enabled() && !haveEnabledInputs && config.Autodiscover == nil && !b.Manager.Enabled() {
		if !b.InSetupCmd {
			return nil, fmt.Errorf("no modules or inputs enabled and configuration reloading disabled. What files do you want me to watch?")
		}

		// in the `setup` command, log this only as a warning
		b.Info.Logger.Warn("Setup called, but no modules enabled.")
	}

	if *once && config.ConfigInput.Enabled() && config.ConfigModules.Enabled() {
		return nil, fmt.Errorf("input configs and --once cannot be used together")
	}

	if config.IsInputEnabled("stdin") && len(enabledInputs) > 1 {
		return nil, fmt.Errorf("stdin requires to be run in exclusive mode, configured inputs: %s", strings.Join(enabledInputs, ", "))
	}

	fb := &Filebeat{
		done:           make(chan struct{}),
		config:         &config,
		moduleRegistry: moduleRegistry,
		pluginFactory:  plugins,
		logger:         b.Info.Logger,
	}

	err = fb.setupPipelineLoaderCallback(b)
	if err != nil {
		return nil, err
	}

	return fb, nil
}

// setupPipelineLoaderCallback sets the callback function for loading pipelines during setup.
func (fb *Filebeat) setupPipelineLoaderCallback(b *beat.Beat) error {
	if b.Config.Output.Name() != "elasticsearch" && !b.Manager.Enabled() {
		fb.logger.Warn(pipelinesWarning)
		return nil
	}

	overwritePipelines := true
	b.OverwritePipelinesCallback = func(esConfig *conf.C) error {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		esClient, err := eslegclient.NewConnectedClient(ctx, esConfig, "Filebeat", fb.logger)
		if err != nil {
			return err
		}

		// When running the subcommand setup, configuration from modules.d directories
		// have to be loaded using cfg.Reloader. Otherwise those configurations are skipped.
		pipelineLoaderFactory := newPipelineLoaderFactory(ctx, b.Config.Output.Config(), fb.logger)
		enableAllFilesets, _ := b.BeatConfig.Bool("config.modules.enable_all_filesets", -1)
		forceEnableModuleFilesets, _ := b.BeatConfig.Bool("config.modules.force_enable_module_filesets", -1)
		filesetOverrides := fileset.FilesetOverrides{
			EnableAllFilesets:         enableAllFilesets,
			ForceEnableModuleFilesets: forceEnableModuleFilesets,
		}

		modulesFactory := fileset.NewSetupFactory(b.Info, pipelineLoaderFactory, filesetOverrides, b.Paths)
		if fb.config.ConfigModules.Enabled() {
			if enableAllFilesets {
				// All module configs need to be loaded to enable all the filesets
				// contained in the modules.  The default glob just loads the enabled
				// ones.  Switching the glob pattern from *.yml to * achieves this.
				origPath, _ := fb.config.ConfigModules.String("path", -1)
				newPath := strings.TrimSuffix(origPath, ".yml")
				_ = fb.config.ConfigModules.SetString("path", -1, newPath)
			}
			modulesLoader := cfgfile.NewReloader(fb.logger.Named("module.reloader"), fb.pipeline, fb.config.ConfigModules, b.Paths)
			modulesLoader.Load(modulesFactory)
		}

		return fb.moduleRegistry.LoadPipelines(esClient, overwritePipelines)
	}
	return nil
}

func (fb *Filebeat) WithOtelFactoryWrapper(wrapper cfgfile.FactoryWrapper) {
	fb.otelStatusFactoryWrapper = wrapper
}

// loadModulesPipelines is called when modules are configured to do the initial
// setup.
func (fb *Filebeat) loadModulesPipelines(b *beat.Beat) error {
	if b.Config.Output.Name() != "elasticsearch" {
		if !b.Manager.Enabled() {
			fb.logger.Warn(pipelinesWarning)
		}
		return nil
	}

	overwritePipelines := fb.config.OverwritePipelines
	if b.InSetupCmd {
		overwritePipelines = true
	}

	// register pipeline loading to happen every time a new ES connection is
	// established
	callback := func(esClient *eslegclient.Connection, _ *logp.Logger) error {
		return fb.moduleRegistry.LoadPipelines(esClient, overwritePipelines)
	}
	_, err := elasticsearch.RegisterConnectCallback(callback)

	return err
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {
	var err error
	config := fb.config

	if b.Manager != nil {
		b.Manager.RegisterDiagnosticHook("input_metrics", "Metrics from active inputs.",
			"input_metrics.json", "application/json", func() []byte {
				data, err := inputmon.MetricSnapshotJSON(b.Monitoring.InputsRegistry())
				if err != nil {
					b.Info.Logger.Warnw("Failed to collect input metric snapshot for Agent diagnostics.", "error", err)
					return []byte(err.Error())
				}
				return data
			})

		b.Manager.RegisterDiagnosticHook(
			"registry",
			"Filebeat's registry",
			"registry.tar.gz",
			"application/octet-stream",
			gzipRegistry(b.Info.Logger, b.Paths))
	}

	if !fb.moduleRegistry.Empty() {
		err = fb.loadModulesPipelines(b)
		if err != nil {
			return err
		}
	}

	waitFinished := newSignalWait()
	waitEvents := newSignalWait()

	// count active events for waiting on shutdown
	reg := b.Monitoring.StatsRegistry()
	wgEvents := &eventCounter{
		count: monitoring.NewInt(reg, "filebeat.events.active"), // Gauge
		added: monitoring.NewUint(reg, "filebeat.events.added"),
		done:  monitoring.NewUint(reg, "filebeat.events.done"),
	}
	finishedLogger := newFinishedLogger(wgEvents)

	registryMigrator := registrar.NewMigrator(config.Registry, fb.logger, b.Paths)
	if err := registryMigrator.Run(); err != nil {
		fb.logger.Errorf("Failed to migrate registry file: %+v", err)
		return err
	}

	// Use context, like normal people do, hooking up to the beat.done channel
	ctx, cn := context.WithCancel(context.Background())
	go func() {
		<-fb.done
		cn()
	}()

	stateStore, err := openStateStore(ctx, b.Info, fb.logger.Named("filebeat"), config.Registry, b.Paths)
	if err != nil {
		fb.logger.Errorf("Failed to open state store: %+v", err)
		return err
	}
	defer stateStore.Close()

	// If notifier is set, configure the listener for output configuration
	// The notifier passes the elasticsearch output configuration down to the Elasticsearch backed state storage
	// in order to allow it fully configure
	if stateStore.notifier != nil {
		b.OutputConfigReloader = reload.ReloadableFunc(func(r *reload.ConfigWithMeta) error {
			outCfg := conf.Namespace{}
			if err := r.Config.Unpack(&outCfg); err != nil || outCfg.Name() != "elasticsearch" {
				fb.logger.Errorf("Failed to unpack the output config: %v", err)
				return nil
			}

			// Create a new config with the output configuration. Since r.Config is a pointer, a copy is required to
			// avoid concurrent map read and write.
			// See https://github.com/elastic/beats/issues/42815
			configCopy, err := conf.NewConfigFrom(outCfg.Config())
			if err != nil {
				fb.logger.Errorf("Failed to create a new config from the output config: %v", err)
				return nil
			}
			stateStore.notifier.Notify(configCopy)
			return nil
		})
	}

	err = filestream.ValidateInputIDs(config.Inputs, fb.logger.Named("input.filestream"))
	if err != nil {
		fb.logger.Errorf("invalid filestream configuration: %+v", err)
		return err
	}

	// Setup registrar to persist state
	registrar, err := registrar.New(stateStore, finishedLogger, config.Registry.FlushTimeout, fb.logger)
	if err != nil {
		fb.logger.Errorf("Could not init registrar: %v", err)
		return err
	}

	// Make sure all events that were published in
	registrarChannel := newRegistrarLogger(registrar)

	// setup event counting for startup and a global common ACKer, such that all events will be
	// routed to the reigstrar after they've been ACKed.
	// Events with Private==nil or the type of private != file.State are directly
	// forwarded to `finishedLogger`. Events from the `logs` input will first be forwarded
	// to the registrar via `registrarChannel`, which finally forwards the events to finishedLogger as well.
	// The finishedLogger decrements the counters in wgEvents after all events have been securely processed
	// by the registry.
	fb.pipeline = withPipelineEventCounter(b.Publisher, wgEvents)
	fb.pipeline = pipetool.WithACKer(fb.pipeline, eventACKer(finishedLogger, registrarChannel))

	// Filebeat by default required infinite retry. Let's configure this for all
	// inputs by default.  Inputs (and InputController) can overwrite the sending
	// guarantees explicitly when connecting with the pipeline.
	fb.pipeline = pipetool.WithDefaultGuarantees(fb.pipeline, beat.GuaranteedSend)

	outDone := make(chan struct{}) // outDone closes down all active pipeline connections
	pipelineConnector := channel.NewOutletFactory(outDone).Create

	inputsLogger := fb.logger.Named("input")
	v2Inputs := fb.pluginFactory(b.Info, inputsLogger, stateStore, b.Paths)
	v2InputLoader, err := v2.NewLoader(inputsLogger, v2Inputs, "type", cfg.DefaultType)
	if err != nil {
		panic(err) // loader detected invalid state.
	}

	var inputTaskGroup unison.TaskGroup
	defer func() {
		_ = inputTaskGroup.Stop()
	}()

	// Store needs to be fully configured at this point
	if err := v2InputLoader.Init(&inputTaskGroup); err != nil {
		fb.logger.Errorf("Failed to initialize the input managers: %v", err)
		return err
	}

	inputLoader := channel.RunnerFactoryWithCommonInputSettings(b.Info, compat.Combine(
		compat.RunnerFactory(inputsLogger, b.Info, b.Monitoring.InputsRegistry(), v2InputLoader),
		input.NewRunnerFactory(pipelineConnector, registrar, fb.done, fb.logger),
	))

	if fb.otelStatusFactoryWrapper != nil {
		inputLoader = fb.otelStatusFactoryWrapper(inputLoader)
	}

	// Create a ES connection factory for dynamic modules pipeline loading
	var pipelineLoaderFactory fileset.PipelineLoaderFactory
	// The pipelineFactory needs a context to control the connections to ES,
	// when the pipelineFactory/ESClient are not needed any more the context
	// must be cancelled. This pipeline factory will be used by the moduleLoader
	// that is run by a crawler, whenever this crawler is stopped we also cancel
	// the context.
	pipelineFactoryCtx, cancelPipelineFactoryCtx := context.WithCancel(context.Background())
	defer cancelPipelineFactoryCtx()
	if b.Config.Output.Name() == "elasticsearch" {
		pipelineLoaderFactory = newPipelineLoaderFactory(pipelineFactoryCtx, b.Config.Output.Config(), fb.logger)
	} else {
		if !b.Manager.Enabled() {
			fb.logger.Warn(pipelinesWarning)
		}
	}
	moduleLoader := fileset.NewFactory(inputLoader, b.Info, pipelineLoaderFactory, config.OverwritePipelines, b.Paths)
	crawler, err := newCrawler(inputLoader, moduleLoader, config.Inputs, fb.done, *once, fb.logger, b.Paths)
	if err != nil {
		fb.logger.Errorf("Could not init crawler: %v", err)
		return err
	}

	// The order of starting and stopping is important. Stopping is inverted to the starting order.
	// The current order is: registrar, publisher, spooler, crawler
	// That means, crawler is stopped first.

	// Start the registrar
	err = registrar.Start()
	if err != nil {
		return fmt.Errorf("Could not start registrar: %w", err) //nolint:staticcheck //Keep old behavior
	}

	// Stopping registrar will write last state
	defer registrar.Stop()

	// Stopping publisher (might potentially drop items)
	defer func() {
		// Closes first the registrar logger to make sure not more events arrive at the registrar
		// registrarChannel must be closed first to potentially unblock (pretty unlikely) the publisher
		registrarChannel.Close()
		close(outDone) // finally close all active connections to publisher pipeline
	}()

	// Wait for all events to be processed or timeout
	defer waitEvents.Wait()

	if config.OverwritePipelines {
		fb.logger.Debug("modules", "Existing Ingest pipelines will be updated")
	}

	err = crawler.Start(fb.pipeline, config.ConfigInput, config.ConfigModules)
	if err != nil {
		crawler.Stop()
		cancelPipelineFactoryCtx()
		return fmt.Errorf("Failed to start crawler: %w", err) //nolint:staticcheck //Keep old behavior
	}

	// If run once, add crawler completion check as alternative to done signal
	if *once {
		runOnce := func() {
			fb.logger.Info("Running filebeat once. Waiting for completion ...")
			crawler.WaitForCompletion()
			fb.logger.Info("All data collection completed. Shutting down.")
		}
		waitFinished.Add(runOnce)
	}

	// Register reloadable list of inputs and modules
	inputs := cfgfile.NewRunnerList(management.DebugK, inputLoader, fb.pipeline, fb.logger)
	b.Registry.MustRegisterInput(inputs)

	modules := cfgfile.NewRunnerList(management.DebugK, moduleLoader, fb.pipeline, fb.logger)

	var adiscover *autodiscover.Autodiscover
	if fb.config.Autodiscover != nil {
		adiscover, err = autodiscover.NewAutodiscover(
			"filebeat",
			fb.pipeline,
			cfgfile.MultiplexedRunnerFactory(
				cfgfile.MatchHasField("module", moduleLoader),
				cfgfile.MatchDefault(inputLoader),
			),
			autodiscover.QueryConfig(),
			config.Autodiscover,
			b.Keystore,
			fb.logger,
		)
		if err != nil {
			return err
		}
	}
	adiscover.Start()

	// We start the manager when all the subsystem are initialized and ready to received events.
	if err := b.Manager.Start(); err != nil {
		return err
	}

	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(fb.done)
	waitFinished.Wait()

	// Stop reloadable lists, autodiscover -> Stop crawler -> stop inputs -> stop harvesters
	// Note: waiting for crawlers to stop here in order to install wgEvents.Wait
	//       after all events have been enqueued for publishing. Otherwise wgEvents.Wait
	//       or publisher might panic due to concurrent updates.
	inputs.Stop()
	modules.Stop()
	adiscover.Stop()
	crawler.Stop()
	cancelPipelineFactoryCtx()

	timeout := fb.config.ShutdownTimeout
	// Checks if on shutdown it should wait for all events to be published
	waitPublished := fb.config.ShutdownTimeout > 0 || *once
	if waitPublished {
		// Wait for registrar to finish writing registry
		waitEvents.Add(withLog(wgEvents.Wait,
			"Continue shutdown: All enqueued events being published.", fb.logger))
		// Wait for either timeout or all events having been ACKed by outputs.
		if fb.config.ShutdownTimeout > 0 {
			fb.logger.Info("Shutdown output timer started. Waiting for max %v.", timeout)
			waitEvents.Add(withLog(waitDuration(timeout),
				"Continue shutdown: Time out waiting for events being published.", fb.logger))
		} else {
			waitEvents.AddChan(fb.done)
		}
	}

	// Stop the manager and stop the connection to any dependent services.
	// The Manager started to have a working implementation when
	// https://github.com/elastic/beats/pull/34416 was merged.
	// This is intended to enable TLS certificates reload on a long
	// running Beat.
	//
	// However calling b.Manager.Stop() here messes up the behavior of the
	// --once flag because it makes Filebeat exit early.
	// So if --once is passed, we don't call b.Manager.Stop().
	if !*once {
		b.Manager.Stop()
	}

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {
	fb.logger.Info("Stopping filebeat")

	// Stop Filebeat
	fb.stopOnce.Do(func() { close(fb.done) })
}

// Create a new pipeline loader (es client) factory
func newPipelineLoaderFactory(ctx context.Context, esConfig *conf.C, logger *logp.Logger) fileset.PipelineLoaderFactory {
	pipelineLoaderFactory := func() (fileset.PipelineLoader, error) {
		esClient, err := eslegclient.NewConnectedClient(ctx, esConfig, "Filebeat", logger)
		if err != nil {
			return nil, fmt.Errorf("Error creating Elasticsearch client: %w", err) //nolint:staticcheck //Keep old behavior
		}
		return esClient, nil
	}
	return pipelineLoaderFactory
}
