package beater

import (
	"flag"
	"fmt"

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

	moduleRegistry, err := fileset.NewModuleRegistry(config.Modules, b.Info.Version, true)
	if err != nil {
		return nil, err
	}
	if !moduleRegistry.Empty() {
		logp.Info("Enabled modules/filesets: %s", moduleRegistry.InfoString())
	}

	moduleProspectors, err := moduleRegistry.GetProspectorConfigs()
	if err != nil {
		return nil, err
	}

	if err := config.FetchConfigs(); err != nil {
		return nil, err
	}

	// Add prospectors created by the modules
	config.Prospectors = append(config.Prospectors, moduleProspectors...)

	haveEnabledProspectors := false
	for _, prospector := range config.Prospectors {
		if prospector.Enabled() {
			haveEnabledProspectors = true
			break
		}
	}

	if !config.ConfigProspector.Enabled() && !config.ConfigModules.Enabled() && !haveEnabledProspectors && config.Autodiscover == nil {
		if !b.InSetupCmd {
			return nil, errors.New("No modules or prospectors enabled and configuration reloading disabled. What files do you want me to watch?")
		}

		// in the `setup` command, log this only as a warning
		logp.Warn("Setup called, but no modules enabled.")
	}

	if *once && config.ConfigProspector.Enabled() && config.ConfigModules.Enabled() {
		return nil, errors.New("prospector configs and -once cannot be used together")
	}

	fb := &Filebeat{
		done:           make(chan struct{}),
		config:         &config,
		moduleRegistry: moduleRegistry,
	}

	// register `setup` callback for ML jobs
	b.SetupMLCallback = func(b *beat.Beat) error {
		return fb.loadModulesML(b)
	}
	return fb, nil
}

// loadModulesPipelines is called when modules are configured to do the initial
// setup.
func (fb *Filebeat) loadModulesPipelines(b *beat.Beat) error {
	if b.Config.Output.Name() != "elasticsearch" {
		logp.Warn(pipelinesWarning)
		return nil
	}

	// register pipeline loading to happen every time a new ES connection is
	// established
	callback := func(esClient *elasticsearch.Client) error {
		return fb.moduleRegistry.LoadPipelines(esClient)
	}
	elasticsearch.RegisterConnectCallback(callback)

	return nil
}

func (fb *Filebeat) loadModulesML(b *beat.Beat) error {
	logp.Debug("machine-learning", "Setting up ML jobs for modules")
	var errs multierror.Errors

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
	if err := fb.moduleRegistry.LoadML(esClient); err != nil {
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

			if err := set.LoadML(esClient); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs.Err()
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
	registrar, err := registrar.New(config.RegistryFile, config.RegistryFlush, finishedLogger)
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
		config.Prospectors,
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

	err = crawler.Start(registrar, config.ConfigProspector, config.ConfigModules, pipelineLoaderFactory)
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
		adapter := NewAutodiscoverAdapter(crawler.ProspectorsFactory, crawler.ModulesFactory)
		adiscover, err = autodiscover.NewAutodiscover("filebeat", adapter, config.Autodiscover)
		if err != nil {
			return err
		}
	}
	adiscover.Start()

	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(fb.done)
	waitFinished.Wait()

	// Stop autodiscover -> Stop crawler -> stop prospectors -> stop harvesters
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
