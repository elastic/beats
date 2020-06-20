package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
)

type app struct {
	log  *logp.Logger
	info beat.Info

	// app settings
	Name      string
	Settings  settings
	rawConfig *common.Config // required for index managemnt setup... should be removed

	// configured subsystems
	statestore  *filebeatStore
	scheduler   *scheduler.Scheduler
	inputLoader *v2.Loader

	// statically configured inputs. To be removed in favor of configuring via agent RPC only. Maybe keep here for testing only.
	inputs []v2.Input
}

func main() {
	rc := run()
	if rc != 0 {
		fmt.Fprintf(os.Stderr, "Exit with error code %v\n", rc)
	} else {
		fmt.Fprintln(os.Stderr, "Exit without error")
	}

	os.Exit(rc)
}

func run() (retcode int) {
	var flags flagsConfig
	if err := flags.parseArgs(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read arguments:\n%v\n", err)
		return 2
	}

	app, err := newApp(filepath.Base(os.Args[0]), flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed with: %v\n", err)
		return 1
	}
	defer app.Cleanup()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Run failed with: %v\n", err)
		return 1
	}

	return 0
}

func newApp(name string, flags flagsConfig) (*app, error) {
	app := &app{Name: name}
	return app, app.init(flags)
}

func (app *app) init(flags flagsConfig) error {
	fmt.Fprintf(os.Stdout, "Start Loading configuration %v\n", app.Name)
	defer fmt.Fprintf(os.Stdout, "Finished Loading configuration %v\n", app.Name)

	if err := app.initSettings(flags); err != nil {
		return err
	}
	if err := app.configure(); err != nil {
		return err
	}

	return nil
}

func (app *app) defaultSettings() settings {
	s := settings{
		// XXX: preconfigure logging to stderr
		Logging: logp.DefaultConfig(logp.SystemdEnvironment),
	}
	s.Logging.Beat = app.Name
	return s
}

func (app *app) initSettings(flags flagsConfig) error {
	paths, err := initPaths(flags.Path)
	if err != nil {
		return fmt.Errorf("can not initialize application paths: %w", err)
	}

	config, err := readConfigFiles(paths, flags.ConfigFiles, flags.StrictPermissions)
	if err != nil {
		return fmt.Errorf("Failed to read config file(s): %w", err)
	}
	app.rawConfig = config

	settings := app.defaultSettings()
	settings.Path = flags.Path
	if err = config.Unpack(&settings); err != nil {
		return fmt.Errorf("failed to unpack the configuration files: %w", err)
	}
	if settings.Path, err = initPaths(settings.Path); err != nil {
		return fmt.Errorf("can not initialize application paths: %w", err)
	}

	if settings.Registry.Path == "" {
		settings.Registry.Path = filepath.Join(settings.Path.Data, "registry")
	}

	app.Settings = settings
	return nil
}

func (app *app) configure() error {
	// logging first!!!
	if err := logp.Configure(app.Settings.Logging); err != nil {
		return fmt.Errorf("failed to initialize logging output: %w", err)
	}
	app.log = logp.NewLogger(app.Name)

	// TODO: make configurable via CLI?
	app.info = beat.Info{
		Beat: app.Name,
		Name: app.Name,
		// Hostname, ID, EphemeralID, missing. should they be set by agent?
		Version: "8.0.0", // XXX: hard coded for now :/
	}

	// configure filebeat store
	ok := false
	fmt.Printf("%#v\n", app.Settings.Registry)
	store, err := openStateStore(app.info, app.log.Named("store"), app.Settings.Registry)
	if err != nil {
		return err
	}
	app.statestore = store
	defer cleanup.IfNot(&ok, func() { store.Close() })

	// configure heartbeat scheduler
	locationName := app.Settings.Location
	if locationName == "" {
		locationName = "Local"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return err
	}
	app.scheduler = scheduler.NewWithLocation(app.Settings.Limits.Monitors, nil, location)

	inputsCollection := registryList{
		withTypePrefix("logs", makeFilebeatRegistry(app.info, app.log, app.statestore)),
		withTypePrefix("monitor", makeHeartbeatRegistry(app.scheduler)),
		withTypePrefix("metrics", makeMetricbeatRegistry(app.info, nil)),
		withTypePrefix("audit", makeAuditbeatRegistry(app.info, nil)),
		withTypePrefix("net", makePacketbeatRegistry()),
	}
	app.inputLoader = v2.NewLoader(app.log, inputsCollection, "type", "")

	// Let's configure inputs. Inputs won't do any processing, yet.
	var inputs []v2.Input
	for _, config := range app.Settings.Inputs {
		if !config.Enabled() {
			continue
		}

		input, err := app.inputLoader.Configure(config)
		if err != nil {
			return fmt.Errorf("Failed to configure inputs: %w", err)
		}
		inputs = append(inputs, input)
	}
	app.inputs = inputs

	ok = true
	return nil
}

func (app *app) Run() error {
	app.log.Infof("Starting... %v", app.Name)

	var autoCancel ctxtool.AutoCancel
	defer autoCancel.Cancel()

	// XXX: A little overkill to init all index management, but makes output setup easier for now
	indexManagement, err := idxmgmt.MakeDefaultSupport(nil)(nil, app.info, app.rawConfig)
	if err != nil {
		return err
	}

	pipeline, err := pipeline.Load(app.info,
		pipeline.Monitors{
			Metrics:   nil,
			Telemetry: monitoring.GetNamespace("state").GetRegistry(),
			Logger:    app.log.Named("publisher"),
			Tracer:    nil,
		},
		app.Settings.Pipeline,
		nil,
		makeOutputFactory(app.info, indexManagement, app.Settings.Output),
	)
	if err != nil {
		return err
	}
	defer pipeline.Close()

	// setup shutdown signal on OS signal
	sigContext := autoCancel.With(osSignalContext(os.Interrupt))
	sigContext = autoCancel.With(ctxtool.WithFunc(sigContext, func() {
		app.log.Info("Shutdown signal received")
	}))

	// setup input lifetime management and shutdown signaling
	inputTaskGroup := unison.TaskGroup{}
	sigContext = autoCancel.With(ctxtool.WithFunc(sigContext, func() {
		app.log.Info("Stopping inputs...")
		if err := inputTaskGroup.Stop(); err != nil {
			app.log.Errorf("input failures detected: %v", err)
		}
		app.log.Info("Inputs stopped.")
	}))

	inputManagerTaskGroup := unison.TaskGroup{StopOnError: func(_ error) bool { return true }}
	sigContext = autoCancel.With(ctxtool.WithFunc(sigContext, func() {
		app.log.Info("Stopping input managers...")
		err := inputManagerTaskGroup.Stop()
		if err != nil {
			app.log.Errorf("Input managers failed: %v", err)
		}
		app.log.Info("Input managers stopped.")
	}))

	// start input managers
	app.log.Info("Starting input managers...")
	if err := app.inputLoader.Init(&inputManagerTaskGroup, v2.ModeRun); err != nil {
		logp.Err("Failed to initialize the input managers: %v", err)
		return err
	}
	app.log.Info("Input management active...")

	// start inputs
	app.log.Info("Starting inputs...")
	inputLogger := app.log.Named("input")
	for _, input := range app.inputs {
		input := input
		inputTaskGroup.Go(func(cancel unison.Canceler) error {
			inputContext := v2.Context{
				Logger:      inputLogger,
				ID:          "to-be-set-by-agent",
				Agent:       app.info,
				Cancelation: cancel,
			}
			return input.Run(inputContext, pipeline)
		})
	}
	app.log.Info("Inputs active...")

	// XXX: heartbeat scheduler.... we start this one last, as this is  how
	// heartbeat itself handles the scheduler.
	if err := app.scheduler.Start(); err != nil {
		return err
	}
	defer app.scheduler.Stop()

	// All inputs running. Wait for shutdown signal
	app.log.Info("Start finished")
	<-sigContext.Done()
	app.log.Info("Shutting down...")

	return nil
}

func (app *app) Cleanup() {
	app.log.Info("Shutting down internal subsytems")
	defer app.log.Info("Finished shutting down internal subsystems")

	app.statestore.Close()
}

func makeOutputFactory(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	cfg common.ConfigNamespace,
) func(outputs.Observer) (string, outputs.Group, error) {
	return func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := createOutput(info, indexManagement, outStats, cfg)
		return cfg.Name(), out, err
	}
}

func createOutput(
	info beat.Info,
	indexManagement idxmgmt.Supporter,
	stats outputs.Observer,
	cfg common.ConfigNamespace,
) (outputs.Group, error) {
	if !cfg.IsSet() {
		return outputs.Group{}, nil
	}
	return outputs.Load(indexManagement, info, stats, cfg.Name(), cfg.Config())
}
