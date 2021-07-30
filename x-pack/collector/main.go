package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/ctxtool/osctx"
	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/collector/internal/adapter/pb"
	"github.com/elastic/beats/v7/x-pack/collector/internal/adapter/registries"
	"github.com/elastic/beats/v7/x-pack/collector/internal/cfgload"
	"github.com/elastic/beats/v7/x-pack/collector/internal/management"
	"github.com/elastic/beats/v7/x-pack/collector/internal/pipeline"
	inputs "github.com/elastic/beats/v7/x-pack/filebeat/input/default-inputs"
)

type app struct {
	log  *logp.Logger
	info beat.Info

	// app settings
	Name           string
	Settings       settings
	inputsRegistry v2.Registry

	// configured subsystems
	statestore    *kvStore
	scheduler     *scheduler.Scheduler
	pipelines     *pipeline.Controller
	agentManager  *management.ConfigManager
	configWatcher *cfgload.Watcher
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
	// setup shutdown signaling based on OS signal handling
	// We shutdown early if a signal is received during setup.
	osSig, cancel := osctx.WithSignal(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	var flags flagsConfig
	if err := flags.parseArgs(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read arguments:\n%v\n", err)
		return 2
	}
	if osSig.Err() != nil {
		return 0
	}

	app, err := newApp(filepath.Base(os.Args[0]), flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed with: %v\n", err)
		return 1
	}
	defer app.Cleanup()
	if osSig.Err() != nil {
		return 0
	}

	if err := app.Run(osSig); err != nil {
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
	if err := app.configure(flags); err != nil {
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

	configReader := &cfgload.Loader{Home: paths.Config, StrictPermissions: flags.StrictPermissions}
	config, err := configReader.ReadFiles(flags.ConfigFiles)
	if err != nil {
		return fmt.Errorf("Failed to read config file(s): %w", err)
	}

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

	if settings.Manager.IsManaged() && flags.Reload {
		return errors.New("config reloading and managed mode must not be enabled together")
	}
	if settings.Manager.IsManaged() && !flags.Reload && len(settings.Pipeline.Inputs) == 0 {
		return errors.New("unmanaged mode requires inputs to be configured")
	}

	app.Settings = settings
	return nil
}

func (app *app) configure(flags flagsConfig) error {
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
	store, err := newKVStore(app.info, app.log.Named("store"), app.Settings.Registry)
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

	app.agentManager, err = management.NewConfigManager(app.log, app.Settings.Manager)
	if err != nil {
		return err
	}

	app.inputsRegistry = makeInputRegistry(app.info, app.log, app.scheduler, app.statestore)
	app.pipelines, err = pipeline.NewController(
		app.log, app.info,
		app.inputsRegistry,
		outputPlugins(app.info),
		app.Settings.Pipeline,
	)
	if err != nil {
		return err
	}

	if flags.Reload {
		app.configWatcher = &cfgload.Watcher{
			Log:    app.log.Named("config-watcher"),
			Files:  flags.ConfigFiles,
			Reader: &cfgload.Loader{Home: app.Settings.Path.Config, StrictPermissions: flags.StrictPermissions},
		}
	}

	ok = true
	return nil
}

func makeInputRegistry(info beat.Info, log *logp.Logger, sched *scheduler.Scheduler, store *kvStore) v2.Registry {
	return registries.Combine(
		// filebeat v2 inputs
		registries.Prefixed("logs", v2.MustPluginRegistry(inputs.Init(info, log.Named("input"), store))),
		// packetbeat as v2 input
		registries.Prefixed("net", v2.MustPluginRegistry([]v2.Plugin{pb.Plugin()})),

		// metricbeat,`auditbeat, heartbeat based on legacy runner factories
		registries.Prefixed("monitor", makeHeartbeatRegistry(info, sched)),
		registries.Prefixed("metrics", makeMetricbeatRegistry()),
	)
}

func (app *app) Run(sigContext context.Context) error {
	app.log.Infof("Starting... %v", app.Name)

	var autoCancel ctxtool.AutoCancel
	defer autoCancel.Cancel()

	sigContext = autoCancel.With(ctxtool.WithFunc(sigContext, func() {
		app.log.Info("Shutdown signal received")
	}))

	// App internal jobs that are required for the app to run correctly are
	// registered with the appTaskGroup.  The app is forced to shut down if an
	// essential subsystem fails.
	appTaskGroup := unison.TaskGroupWithCancel(sigContext)
	appTaskGroup.OnQuit = func(err error) (unison.TaskGroupStopAction, error) {
		if err == context.Canceled {
			return unison.TaskGroupStopActionContinue, nil
		}

		debug.PrintStack()
		app.log.Errorf("Critical error, forcing shutdown: %+v", err)
		autoCancel.Cancel()
		return unison.TaskGroupStopActionShutdown, err
	}

	// Start inputs and input managers. Input managers are essential to inputs. If an Input Manager fails,
	// all inputs are stopped and the task returns an error, which is finally propagate to other
	// subsystems.
	// Inputs are not  allowed to be active if the inputs managers are not
	// running. This requires us to propage to wait for all inputs/outputs to be stopped, before we can propagate the
	// the shutdown signal to the input managers.
	appTaskGroup.Go(func(cancel context.Context) error {
		inputManagerTaskGroup := unison.TaskGroupWithCancel(cancel)
		inputManagerTaskGroup.OnQuit = unison.StopOnError
		defer inputManagerTaskGroup.Stop()

		// Start input managers in background. We shutdown if any background task failed
		app.log.Info("Starting input managers...")
		if err := app.inputsRegistry.Init(inputManagerTaskGroup, v2.ModeRun); err != nil {
			logp.Err("Failed to initialize the input managers: %v", err)
			return err
		}
		app.log.Info("Input management active...")
		return app.pipelines.Run(inputManagerTaskGroup.Context())
	})

	// esatblish connection with agent for status reporing and dynamic configuration updates.
	// the manager should be the last subsystem that is shutdown.
	if app.agentManager != nil {
		appTaskGroup.Go(func(cancel context.Context) error {
			return app.agentManager.Run(cancel, management.EventHandler{
				OnStop:   autoCancel.Cancel,
				OnConfig: app.onConfig,
			})
		})
	}

	if app.configWatcher != nil {
		appTaskGroup.Go(func(cancel context.Context) error {
			return app.configWatcher.Run(cancel, app.onConfig)
		})
	}

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

func (app *app) onConfig(cfg *common.Config) error {
	var settings dynamicSettings
	if err := cfg.Unpack(&settings); err != nil {
		return err
	}
	return app.pipelines.UpdateConfig(settings.Pipeline)
}

func (app *app) Cleanup() {
	app.log.Info("Shutting down internal subsytems")
	defer app.log.Info("Finished shutting down internal subsystems")

	app.statestore.Close()
}
