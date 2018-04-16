package beater

import (
	"flag"
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/setup/kibana"

	fbautodiscover "github.com/elastic/beats/filebeat/autodiscover"
	"github.com/elastic/beats/filebeat/channel"
	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/filebeat/registrar"

	// Add filebeat level processors
	_ "github.com/elastic/beats/filebeat/processor/add_kubernetes_metadata"
)

const pipelinesWarning = "Filebeat is unable to load the Ingest Node pipelines for the configured" +
	" modules because the Elasticsearch output is not configured/enabled. If you have" +
	" already loaded the Ingest Node pipelines or are using Logstash pipelines, you" +
	" can ignore this warning."

var (
	once = flag.Bool("once", false, "Run filebeat only once until all harvesters reach EOF")
)

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config         *cfg.Config
	moduleRegistry *fileset.ModuleRegistry
	done           chan struct{}
}

// New creates a new Filebeat pointer instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	config := cfg.DefaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	err := cfgwarn.CheckRemoved5xSettings(rawConfig, "spool_size", "publish_async", "idle_timeout")
	if err != nil {
		return nil, err
	}

	if len(config.Prospectors) > 0 {
		cfgwarn.Deprecate("7.0.0", "prospectors are deprecated, Use `inputs` instead.")
		if len(config.Inputs) > 0 {
			return nil, fmt.Errorf("prospectors and inputs used in the configuration file, define only inputs not both")
		}
		config.Inputs = config.Prospectors
	}

	if config.ConfigProspector != nil {
		cfgwarn.Deprecate("7.0.0", "config.prospectors are deprecated, Use `config.inputs` instead.")
		if config.ConfigInput != nil {
			return nil, fmt.Errorf("config.prospectors and config.inputs used in the configuration file, define only config.inputs not both")
		}
		config.ConfigInput = config.ConfigProspector
	}

	moduleRegistry, err := fileset.NewModuleRegistry(config.Modules, b.Info.Version, true)
	if err != nil {
		return nil, err
	}
	if !moduleRegistry.Empty() {
		logp.Info("Enabled modules/filesets: %s", moduleRegistry.InfoString())
	}

	moduleInputs, err := moduleRegistry.GetInputConfigs()
	if err != nil {
		return nil, err
	}

	if err := config.FetchConfigs(); err != nil {
		return nil, err
	}

	// Add inputs created by the modules
	config.Inputs = append(config.Inputs, moduleInputs...)

	enabledInputs := config.ListEnabledInputs()
	var haveEnabledInputs bool
	if len(enabledInputs) > 0 {
		haveEnabledInputs = true
	}

	if !config.ConfigInput.Enabled() && !config.ConfigModules.Enabled() && !haveEnabledInputs && config.Autodiscover == nil {
		if !b.InSetupCmd {
			return nil, errors.New("no modules or inputs enabled and configuration reloading disabled. What files do you want me to watch?")
		}

		// in the `setup` command, log this only as a warning
		logp.Warn("Setup called, but no modules enabled.")
	}

	if *once && config.ConfigInput.Enabled() && config.ConfigModules.Enabled() {
		return nil, errors.New("input configs and -once cannot be used together")
	}

	if config.IsInputEnabled("stdin") && len(enabledInputs) > 1 {
		return nil, fmt.Errorf("stdin requires to be run in exclusive mode, configured inputs: %s", strings.Join(enabledInputs, ", "))
	}

	fb := &Filebeat{
		done:           make(chan struct{}),
		config:         &config,
		moduleRegistry: moduleRegistry,
	}

	// register `setup` callback for ML jobs
	b.SetupMLCallback = func(b *beat.Beat, kibanaConfig *common.Config) error {
		return fb.loadModulesML(b, kibanaConfig)
	}

	err = fb.setupPipelineLoaderCallback(b)
	if err != nil {
		return nil, err
	}

	return fb, nil
}

func (fb *Filebeat) setupPipelineLoaderCallback(b *beat.Beat) error {
	if !fb.moduleRegistry.Empty() {
		overwritePipelines := fb.config.OverwritePipelines
		if b.InSetupCmd {
			overwritePipelines = true
		}

		b.OverwritePipelinesCallback = func(esConfig *common.Config) error {
			esClient, err := elasticsearch.NewConnectedClient(esConfig)
			if err != nil {
				return err
			}
			return fb.moduleRegistry.LoadPipelines(esClient, overwritePipelines)
		}
	}
	return nil
}

// loadModulesPipelines is called when modules are configured to do the initial
// setup.
func (fb *Filebeat) loadModulesPipelines(b *beat.Beat) error {
	if b.Config.Output.Name() != "elasticsearch" {
		logp.Warn(pipelinesWarning)
		return nil
	}

	overwritePipelines := fb.config.OverwritePipelines
	if b.InSetupCmd {
		overwritePipelines = true
	}

	// register pipeline loading to happen every time a new ES connection is
	// established
	callback := func(esClient *elasticsearch.Client) error {
		return fb.moduleRegistry.LoadPipelines(esClient, overwritePipelines)
	}
	elasticsearch.RegisterConnectCallback(callback)

	return nil
}

func (fb *Filebeat) loadModulesML(b *beat.Beat, kibanaConfig *common.Config) error {
	var errs multierror.Errors

	logp.Debug("machine-learning", "Setting up ML jobs for modules")

	if b.Config.Output.Name() != "elasticsearch" {
		logp.Warn("Filebeat is unable to load the Xpack Machine Learning configurations for the" +
			" modules because the Elasticsearch output is not configured/enabled.")
		return nil
	}

	esConfig := b.Config.Output.Config()
	esClient, err := elasticsearch.NewConnectedClient(esConfig)
	if err != nil {
		return errors.Errorf("Error creating Elasticsearch client: %v", err)
	}

	if kibanaConfig == nil {
		kibanaConfig = common.NewConfig()
	}

	kibanaClient, err := kibana.NewKibanaClient(kibanaConfig)
	if err != nil {
		return errors.Errorf("Error creating Kibana client: %v", err)
	}

	kibanaVersion, err := common.NewVersion(kibanaClient.GetVersion())
	if err != nil {
		return err
	}

	if err := setupMLBasedOnVersion(fb.moduleRegistry, esClient, kibanaClient, kibanaVersion); err != nil {
		errs = append(errs, err)
	}

	// Add dynamic modules.d
	if fb.config.ConfigModules.Enabled() {
		config := cfgfile.DefaultDynamicConfig
		fb.config.ConfigModules.Unpack(&config)

		modulesManager, err := cfgfile.NewGlobManager(config.Path, ".yml", ".disabled")
		if err != nil {
			return errors.Wrap(err, "initialization error")
		}

		for _, file := range modulesManager.ListEnabled() {
			confs, err := cfgfile.LoadList(file.Path)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "error loading config file"))
				continue
			}
			set, err := fileset.NewModuleRegistry(confs, "", false)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if err := setupMLBasedOnVersion(set, esClient, kibanaClient, kibanaVersion); err != nil {
				errs = append(errs, err)
			}

		}
	}

	return errs.Err()
}

func setupMLBasedOnVersion(reg *fileset.ModuleRegistry, esClient *elasticsearch.Client, kibanaClient *kibana.Client, kibanaVersion *common.Version) error {
	if isElasticsearchLoads(kibanaVersion) {
		return reg.LoadML(esClient)
	}
	return reg.SetupML(esClient, kibanaClient)
}

func isElasticsearchLoads(kibanaVersion *common.Version) bool {
	if kibanaVersion.Major < 6 || kibanaVersion.Major == 6 && kibanaVersion.Minor < 1 {
		return true
	}
	return false
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {
	var err error
	config := fb.config

	if !fb.moduleRegistry.Empty() {
		err = fb.loadModulesPipelines(b)
		if err != nil {
			return err
		}
	}

	waitFinished := newSignalWait()
	waitEvents := newSignalWait()

	// count active events for waiting on shutdown
	wgEvents := &eventCounter{
		count: monitoring.NewInt(nil, "filebeat.events.active"),
		added: monitoring.NewUint(nil, "filebeat.events.added"),
		done:  monitoring.NewUint(nil, "filebeat.events.done"),
	}
	finishedLogger := newFinishedLogger(wgEvents)

	// Setup registrar to persist state
	registrar, err := registrar.New(config.RegistryFile, config.RegistryFilePermissions, config.RegistryFlush, finishedLogger)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	// Make sure all events that were published in
	registrarChannel := newRegistrarLogger(registrar)

	err = b.Publisher.SetACKHandler(beat.PipelineACKHandler{
		ACKEvents: newEventACKer(registrarChannel).ackEvents,
	})
	if err != nil {
		logp.Err("Failed to install the registry with the publisher pipeline: %v", err)
		return err
	}

	outDone := make(chan struct{}) // outDone closes down all active pipeline connections
	crawler, err := crawler.New(
		channel.NewOutletFactory(outDone, b.Publisher, wgEvents).Create,
		config.Inputs,
		b.Info.Version,
		fb.done,
		*once)
	if err != nil {
		logp.Err("Could not init crawler: %v", err)
		return err
	}

	// The order of starting and stopping is important. Stopping is inverted to the starting order.
	// The current order is: registrar, publisher, spooler, crawler
	// That means, crawler is stopped first.

	// Start the registrar
	err = registrar.Start()
	if err != nil {
		return fmt.Errorf("Could not start registrar: %v", err)
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

	// Create a ES connection factory for dynamic modules pipeline loading
	var pipelineLoaderFactory fileset.PipelineLoaderFactory
	if b.Config.Output.Name() == "elasticsearch" {
		pipelineLoaderFactory = newPipelineLoaderFactory(b.Config.Output.Config())
	} else {
		logp.Warn(pipelinesWarning)
	}

	if config.OverwritePipelines {
		logp.Debug("modules", "Existing Ingest pipelines will be updated")
	}

	err = crawler.Start(registrar, config.ConfigInput, config.ConfigModules, pipelineLoaderFactory, config.OverwritePipelines)
	if err != nil {
		crawler.Stop()
		return err
	}

	// If run once, add crawler completion check as alternative to done signal
	if *once {
		runOnce := func() {
			logp.Info("Running filebeat once. Waiting for completion ...")
			crawler.WaitForCompletion()
			logp.Info("All data collection completed. Shutting down.")
		}
		waitFinished.Add(runOnce)
	}

	var adiscover *autodiscover.Autodiscover
	if fb.config.Autodiscover != nil {
		adapter := fbautodiscover.NewAutodiscoverAdapter(crawler.InputsFactory, crawler.ModulesFactory)
		adiscover, err = autodiscover.NewAutodiscover("filebeat", adapter, config.Autodiscover)
		if err != nil {
			return err
		}
	}
	adiscover.Start()

	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(fb.done)
	waitFinished.Wait()

	// Stop autodiscover -> Stop crawler -> stop inputs -> stop harvesters
	// Note: waiting for crawlers to stop here in order to install wgEvents.Wait
	//       after all events have been enqueued for publishing. Otherwise wgEvents.Wait
	//       or publisher might panic due to concurrent updates.
	adiscover.Stop()
	crawler.Stop()

	timeout := fb.config.ShutdownTimeout
	// Checks if on shutdown it should wait for all events to be published
	waitPublished := fb.config.ShutdownTimeout > 0 || *once
	if waitPublished {
		// Wait for registrar to finish writing registry
		waitEvents.Add(withLog(wgEvents.Wait,
			"Continue shutdown: All enqueued events being published."))
		// Wait for either timeout or all events having been ACKed by outputs.
		if fb.config.ShutdownTimeout > 0 {
			logp.Info("Shutdown output timer started. Waiting for max %v.", timeout)
			waitEvents.Add(withLog(waitDuration(timeout),
				"Continue shutdown: Time out waiting for events being published."))
		} else {
			waitEvents.AddChan(fb.done)
		}
	}

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {
	logp.Info("Stopping filebeat")

	// Stop Filebeat
	close(fb.done)
}

// Create a new pipeline loader (es client) factory
func newPipelineLoaderFactory(esConfig *common.Config) fileset.PipelineLoaderFactory {
	pipelineLoaderFactory := func() (fileset.PipelineLoader, error) {
		esClient, err := elasticsearch.NewConnectedClient(esConfig)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating Elasticsearch client")
		}
		return esClient, nil
	}
	return pipelineLoaderFactory
}
