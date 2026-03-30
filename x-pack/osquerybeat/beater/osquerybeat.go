// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/osquery/osquery-go"
	kconfig "github.com/osquery/osquery-go/plugin/config"
	klogger "github.com/osquery/osquery-go/plugin/logger"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/proc"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
	installartifact "github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install/artifact"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pub"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var (
	ErrInvalidQueryConfig = errors.New("invalid query configuration")
	ErrAlreadyRunning     = errors.New("already running")
	ErrQueryExecution     = errors.New("failed query execution")
	ErrActionRequest      = errors.New("invalid action request")
	ErrOsquerydExited     = errors.New("osqueryd exited")
)

const (
	adhocOsqueriesTypesCacheSize = 256 // The final cache size equals the number of periodic queries plus this value, in order to have additional cache for ad-hoc queries

	// The interval in second for configuration refresh;
	// osqueryd child process requests configuration from the configuration plugin implemented in osquerybeat
	configurationRefreshIntervalSecs = 60

	osqueryTimeout    = 1 * time.Minute
	osqueryMaxTimeout = 24 * time.Hour
)

const (
	osqueryInputType     = "osquery"
	extManagerServerName = "osqextman"
	configPluginName     = "osq_config"
	loggerPluginName     = "osq_logger"

	// scheduledQueryProfilesDiagTimeout is the timeout for the scheduled_query_profiles diagnostic hook.
	// Large schedules may need a longer timeout; increase if the diagnostic returns incomplete data.
	scheduledQueryProfilesDiagTimeout = 20 * time.Second
)

// osquerybeat configuration.
type osquerybeat struct {
	b      *beat.Beat
	config config.Config
	// osquery install settings are sourced from inputs[0].osquery.elastic_options.install.
	osqueryInstallConfig config.InstallConfig
	// runtime-selected osquery metadata.
	osqueryVersion string
	osquerySource  string

	pub          osquerybeatPublisher
	qp           *queryProfiler
	liveProfiles *liveProfileStore

	log *logp.Logger

	// Beat lifecycle context, cancelled on Stop
	cancel context.CancelFunc
	mx     sync.Mutex

	diagMx        sync.RWMutex
	diagQueryExec queryExecutor

	// parent process watcher
	watcher *Watcher

	osquerydFactory osqd.RunnerFactory
	executablePath  func() (string, error)
}

type osquerybeatPublisher interface {
	scheduledQueryPublisher
	actionResultPublisher
	Configure(inputs []config.InputConfig) error
	Close()
}

var _ osquerybeatPublisher = (*pub.Publisher)(nil)

// New creates an instance of osquerybeat.
func New(b *beat.Beat, cfg *conf.C) (beat.Beater, error) {
	log := logp.NewLogger("osquerybeat")

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	installCfg := config.GetOsqueryInstallConfig(c.Inputs)
	if err := installCfg.NormalizeAndValidate(); err != nil {
		return nil, fmt.Errorf("invalid osquery.elastic_options.install configuration: %w", err)
	}

	bt := &osquerybeat{
		b:                    b,
		config:               c,
		osqueryInstallConfig: installCfg,
		log:                  log,
		pub:                  pub.New(b, log),
		qp:                   newQueryProfiler(log),
		osquerydFactory:      osqd.New,
		executablePath:       os.Executable,
	}

	profileCfg := config.GetQueryProfileStorageConfig(c.Inputs)
	if profileCfg.EnabledOrDefault() {
		profileDir := b.Paths.Resolve(paths.Data, filepath.Join("osquerybeat", "live_query_profiles"))
		store, err := newLiveProfileStore(log, profileDir, profileCfg.MaxProfilesOrDefault())
		if err != nil {
			log.Warnw("failed to initialize live query profile storage", "error", err)
		} else {
			bt.liveProfiles = store
		}
	}

	return bt, nil
}

func (bt *osquerybeat) init() (context.Context, error) {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.cancel != nil {
		return nil, ErrAlreadyRunning
	}
	var ctx context.Context
	ctx, bt.cancel = context.WithCancel(context.Background())

	if bt.watcher != nil {
		bt.watcher.Close()
	}
	bt.watcher = NewWatcher(bt.log)
	return ctx, nil
}

func (bt *osquerybeat) close() {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.pub != nil {
		bt.pub.Close()
	}
	if bt.cancel != nil {
		bt.cancel()
		bt.cancel = nil
	}

	// Start watching the parent process.
	// The beat exits if the process gets orphaned.
	if bt.watcher != nil {
		go bt.watcher.Run()
		bt.watcher = nil
	}
}

// Run starts osquerybeat.
func (bt *osquerybeat) Run(b *beat.Beat) error {
	pj, err := proc.CreateJobObject()
	if err != nil {
		return fmt.Errorf("failed to create process JobObject: %w", err)
	}
	defer pj.Close()

	ctx, err := bt.init()
	if err != nil {
		return err
	}
	defer bt.close()

	// Watch input configuration updates
	inputConfigCh := config.WatchInputs(ctx, bt.log, b.Registry)

	// Create socket path
	socketPath, cleanupFn, err := osqd.CreateSocketPath()
	if err != nil {
		b.Manager.UpdateStatus(status.Failed, "Failed to create socket path: "+err.Error())
		return err
	}
	defer cleanupFn()

	osqueryRuntime, err := bt.resolveOsqueryRuntime(ctx)
	if err != nil {
		b.Manager.UpdateStatus(status.Failed, "Failed to resolve osquery runtime: "+err.Error())
		return err
	}
	bt.osqueryVersion = osqueryRuntime.Version
	bt.osquerySource = osqueryRuntime.Source
	bt.log.Infof("using osquery runtime source=%s version=%s", bt.osquerySource, bt.osqueryVersion)

	opts := []osqd.Option{
		osqd.WithLogger(bt.log),
		osqd.WithConfigRefresh(configurationRefreshIntervalSecs),
		osqd.WithConfigPlugin(configPluginName),
		osqd.WithLoggerPlugin(loggerPluginName),
	}
	if osqueryRuntime.BinDir != "" {
		opts = append(opts, osqd.WithBinaryPath(osqueryRuntime.BinDir))
	}
	if osqueryRuntime.ExtensionPath != "" {
		opts = append(opts, osqd.WithExtensionPath(osqueryRuntime.ExtensionPath))
	}

	// Create osqueryd runner using factory
	osq, err := bt.osquerydFactory(
		socketPath,
		opts...,
	)

	if err != nil {
		b.Manager.UpdateStatus(status.Failed, "Failed to create osqueryd: "+err.Error())
		return err
	}

	// Check that osqueryd exists and runnable
	err = osq.Check(ctx)
	if err != nil {
		b.Manager.UpdateStatus(status.Failed, "Failed to check osqueryd: "+err.Error())
		return err
	}

	// Initialize osqueryd health monitoring
	osqdMetrics := newOsquerydMetrics(bt.b.Monitoring.StatsRegistry(), bt.log)

	// Set reseable action handler
	rah := newResetableActionHandler(bt.pub, bt.log)
	defer rah.Clear()

	g, ctx := errgroup.WithContext(ctx)

	// Start osquery runner.
	// It restarts osquery on configuration options change
	// It exits if osqueryd fails to run for any reason, like a bad configuration for example
	runner := newOsqueryRunner(bt.log)
	g.Go(func() error {
		return runner.Run(ctx, func(ctx context.Context, flags osqd.Flags, inputCh <-chan []config.InputConfig) error {
			return bt.runOsquery(ctx, b, osq, flags, inputCh, rah, osqdMetrics)
		})
	})

	// Start osquery only if config has inputs, otherwise it will be started on the first configuration sent from the agent
	// This way we don't need to persist the configuration for configuration plugin, because osquery is not running until
	// we have the first valid configuration
	if len(bt.config.Inputs) > 0 {
		_ = runner.Update(ctx, bt.config.Inputs)
	}

	// Ensure that all the hooks and actions are ready before starting the Manager
	// to receive configuration.
	bt.registerDiagnosticHooks(b)
	if err := b.Manager.Start(); err != nil {
		b.Manager.UpdateStatus(status.Failed, "Failed to start manager: "+err.Error())
		return err
	}
	defer b.Manager.Stop()

	// Set the osquery beat version to the manager payload. This allows the bundled osquery version to be reported to the stack.
	bt.setManagerPayload(b)

	// Run main loop
	g.Go(func() error {
		b.Manager.UpdateStatus(status.Configuring, "Initial configuration")
		// Configure publisher from initial input
		err := bt.pub.Configure(bt.config.Inputs)
		if err != nil {
			return err
		}

		for {
			b.Manager.UpdateStatus(status.Running, "Running")
			select {
			case <-ctx.Done():
				b.Manager.UpdateStatus(status.Stopping, "Context cancelled, stopping")
				bt.log.Info("osquerybeat context cancelled, exiting")
				return ctx.Err()
			case inputConfigs := <-inputConfigCh:
				b.Manager.UpdateStatus(status.Configuring, "Received updated configuration")
				err = bt.pub.Configure(inputConfigs)
				if err != nil {
					bt.log.Errorf("Failed to connect beat publisher client, err: %v", err)
					return err
				}
				err = runner.Update(ctx, inputConfigs)
				if err != nil {
					bt.log.Errorf("Failed to configure osquery runner, err: %v", err)
				}
			}
		}
	})

	// Wait for clean exit
	err = g.Wait()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			b.Manager.UpdateStatus(status.Stopped, "Stopped")
			bt.log.Debugf("osquerybeat Run exited, context cancelled")
		} else {
			b.Manager.UpdateStatus(status.Failed, "Failed: "+err.Error())
			bt.log.Errorf("osquerybeat Run exited with error: %v", err)
		}
	} else {
		b.Manager.UpdateStatus(status.Stopped, "Stopped")
		bt.log.Debugf("osquerybeat Run exited")
	}
	return err
}

func (bt *osquerybeat) registerDiagnosticHooks(b *beat.Beat) {
	if b == nil || b.Manager == nil {
		return
	}

	b.Manager.RegisterDiagnosticHook(
		"scheduled_query_profiles",
		"Recent scheduled query profiles collected from osquery_schedule.",
		"scheduled_query_profiles.json",
		"application/json",
		func() []byte {
			ctx, cancel := context.WithTimeout(context.Background(), scheduledQueryProfilesDiagTimeout)
			defer cancel()

			payload := map[string]interface{}{
				"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
			}

			bt.diagMx.RLock()
			scheduledPayload, err := bt.qp.scheduledProfilesDiagnosticsPayload(ctx, bt.diagQueryExec)
			bt.diagMx.RUnlock()
			if err != nil {
				payload["error"] = err.Error()
			} else {
				for key, value := range scheduledPayload {
					payload[key] = value
				}
			}

			liveProfiles := []map[string]interface{}{}
			if bt.liveProfiles != nil {
				liveProfiles = bt.liveProfiles.List()
			}
			payload["live_query_profiles"] = liveProfiles
			payload["live_query_profiles_count"] = len(liveProfiles)

			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				if bt.log != nil {
					bt.log.Warnw("Failed to collect query profiles diagnostics.", "error", err)
				}
				return diagnosticsErrorJSON(err.Error())
			}
			return data
		},
	)
}

func (bt *osquerybeat) setDiagnosticsQueryExecutor(qe queryExecutor) {
	bt.diagMx.Lock()
	defer bt.diagMx.Unlock()
	bt.diagQueryExec = qe
}

func (bt *osquerybeat) getDiagnosticsQueryExecutor() queryExecutor {
	bt.diagMx.RLock()
	defer bt.diagMx.RUnlock()
	return bt.diagQueryExec
}

func (bt *osquerybeat) runOsquery(ctx context.Context, b *beat.Beat, osq osqd.Runner, flags osqd.Flags, inputCh <-chan []config.InputConfig, rah *resetableActionHandler, osqdMetrics *osquerydMetrics) error {
	socketPath := osq.SocketPath()

	// Create a cache for queries types resolution
	cache, err := lru.New[string, map[string]string](adhocOsqueriesTypesCacheSize)
	if err != nil {
		bt.log.Errorf("Failed to create osquery query results types cache: %v", err)
		return err
	}

	// Start osqueryd
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := osq.Run(ctx, flags)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				bt.log.Errorf("Osqueryd exited: %v", err)
			} else {
				bt.log.Errorf("Failed to run osqueryd: %v", err)
			}
		} else {
			// When osqueryd is killed for example there is no error returned
			// but we can't continue running. Exiting.
			bt.log.Info("osqueryd process exited")
			err = ErrOsquerydExited
		}
		return err
	})

	// Create osqueryd client
	cli := osqdcli.New(socketPath,
		osqdcli.WithLogger(bt.log),
		osqdcli.WithTimeout(osqueryTimeout),
		osqdcli.WithMaxTimeout(osqueryMaxTimeout),
		osqdcli.WithCache(cache, adhocOsqueriesTypesCacheSize),
	)

	// Create osquery configuration plugin that loads a persisted configuration from the disk
	configPlugin := NewConfigPlugin(bt.log)
	// Resize cache
	cache.Resize(configPlugin.Count())

	// Create osquery logger plugin
	loggerPlugin := NewLoggerPlugin(bt.log, func(res QueryResult) {
		bt.handleQueryResult(ctx, cli, configPlugin, res)
	})

	// Run main loop
	g.Go(func() error {
		// Connect to osqueryd
		err = cli.Connect(ctx)
		if err != nil {
			return err
		}
		bt.setDiagnosticsQueryExecutor(cli)
		defer cli.Close()
		defer bt.setDiagnosticsQueryExecutor(nil)

		// Start osqueryd health monitoring after connection is established
		g.Go(func() error {
			monitorOsquerydHealth(ctx, cli, osqdMetrics, bt.log)
			return nil
		})

		// Run extensions only after successful connect, otherwise the extension server fails with windows pipes if the pipe was not created by osqueryd yet
		g.Go(func() error {
			return runExtensionServer(ctx, socketPath, configPlugin, loggerPlugin, osqueryTimeout)
		})

		// Register action handler
		bt.registerActionHandler(b, cli, configPlugin, rah)
		defer bt.unregisterActionHandler(b, rah)

		// Process input
		for {
			select {
			case <-ctx.Done():
				bt.log.Info("runOsquery context cancelled, exiting")
				return ctx.Err()
			case inputConfigs := <-inputCh:
				err = configPlugin.Set(inputConfigs)
				if err != nil {
					bt.log.Errorf("failed to set configuration from inputs: %v", err)
					return err
				}
				cache.Resize(configPlugin.Count())
			}
		}
	})

	err = g.Wait()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			bt.log.Debugf("runOsquery exited, context cancelled")
		} else {
			bt.log.Errorf("runOsquery exited with error: %v", err)
		}
	} else {
		bt.log.Debugf("runOsquery exited")
	}
	return err
}

func runExtensionServer(ctx context.Context, socketPath string, configPlugin *ConfigPlugin, loggerPlugin *LoggerPlugin, timeout time.Duration) (err error) {
	// Register config and logger extensions
	extserver, err := osquery.NewExtensionManagerServer(extManagerServerName, socketPath, osquery.ServerTimeout(timeout))
	if err != nil {
		return err
	}

	// Register osquery configuration plugin
	extserver.RegisterPlugin(kconfig.NewPlugin(configPluginName, configPlugin.GenerateConfig))
	// Register osquery logger plugin
	extserver.RegisterPlugin(klogger.NewPlugin(loggerPluginName, loggerPlugin.Log))

	g, ctx := errgroup.WithContext(ctx)
	// Run extension server
	g.Go(func() error {
		return extserver.Run()
	})

	// Run extension server shutdown goroutine, otherwise it waits for ping failure
	g.Go(func() error {
		<-ctx.Done()
		return extserver.Shutdown(context.Background())
	})

	return g.Wait()
}

// nativeScheduleExecutionCount returns the 1-based execution count for a native (interval) schedule,
// computed from start_date and interval so it is deterministic across agents.
// Returns 0 if startDate is empty, interval <= 0, or runTime is before startDate.
func nativeScheduleExecutionCount(startDateRFC3339 string, intervalSecs int, runTimeUnix int64) int64 {
	if startDateRFC3339 == "" || intervalSecs <= 0 {
		return 0
	}
	startTime, err := time.Parse(time.RFC3339, startDateRFC3339)
	if err != nil {
		return 0
	}

	startUnix := startTime.Unix()
	if runTimeUnix < startUnix {
		return 0
	}

	elapsedSeconds := runTimeUnix - startUnix
	return 1 + (elapsedSeconds / int64(intervalSecs))
}

// nativePlannedScheduleTime returns the intended schedule slot for a native interval schedule.
// Falls back to runTimeUnix when schedule metadata is missing or invalid.
func nativePlannedScheduleTime(startDateRFC3339 string, intervalSecs int, runTimeUnix int64) time.Time {
	runTime := time.Unix(runTimeUnix, 0).UTC()
	executionCount := nativeScheduleExecutionCount(startDateRFC3339, intervalSecs, runTimeUnix)
	if executionCount <= 0 {
		return runTime
	}

	startTime, err := time.Parse(time.RFC3339, startDateRFC3339)
	if err != nil {
		return runTime
	}

	return startTime.UTC().Add(time.Duration(executionCount-1) * time.Duration(intervalSecs) * time.Second)
}

func (bt *osquerybeat) handleQueryResult(ctx context.Context, cli *osqdcli.Client, configPlugin *ConfigPlugin, res QueryResult) {
	ns, ok := configPlugin.LookupNamespace(res.Name)
	if !ok {
		bt.log.Debugf("failed to lookup query namespace: %s, the query was possibly removed recently from the schedule", res.Name)
		// Drop the scheduled query results since at this point we don't have the namespace for the datastream where to send the results to
		// and the API key would not have permissions for that namespaces datastream to create the index
		return
	}

	qi, ok := configPlugin.LookupQueryInfo(res.Name)
	if !ok {
		bt.log.Errorf("failed to lookup query info: %s", res.Name)
		return
	}

	// Use policy schedule_id when set, otherwise query name.
	scheduleID := qi.ScheduleID
	if scheduleID == "" {
		scheduleID = res.Name
	}
	// Schedule execution count from start_date + interval (same across agents)
	scheduleExecutionCount := nativeScheduleExecutionCount(qi.StartDate, qi.Interval, res.UnixTime)

	var totalHits int

	responseID := uuid.Must(uuid.NewV4()).String()
	runTime := time.Unix(res.UnixTime, 0)
	plannedScheduleTime := nativePlannedScheduleTime(qi.StartDate, qi.Interval, res.UnixTime)
	publishResolved := func(resultType, action string, hits []map[string]interface{}) {
		totalHits += len(hits)
		meta := queryResultMeta(resultType, action, res, scheduleExecutionCount, plannedScheduleTime)
		bt.pub.Publish(config.Datastream(ns), scheduleID, "schedule_id", responseID, qi.SpaceID, qi.PackID, meta, hits, qi.ECSMapping, nil)
	}

	if res.Action == "snapshot" {
		snapshot, err := cli.ResolveResult(ctx, qi.Query, res.Hits)
		if err != nil {
			bt.log.Errorf("failed to resolve snapshot query result types: %s", res.Name)
			return
		}
		publishResolved("snapshot", "", snapshot)
	} else {
		if len(res.DiffResults.Added) > 0 {
			added, err := cli.ResolveResult(ctx, qi.Query, res.DiffResults.Added)
			if err != nil {
				bt.log.Errorf(`failed to resolve diff query "added" result types: %s`, res.Name)
				return
			}
			publishResolved("diff", "added", added)
		}
		if len(res.DiffResults.Removed) > 0 {
			removed, err := cli.ResolveResult(ctx, qi.Query, res.DiffResults.Removed)
			if err != nil {
				bt.log.Errorf(`failed to resolve diff query "removed" result types: %s`, res.Name)
				return
			}
			publishResolved("diff", "removed", removed)
		}
	}

	if configPlugin.LookupQueryProfile(res.Name) {
		profile, err := bt.qp.profileScheduledQuery(ctx, cli, res.Name)
		if err != nil {
			bt.log.Debugf("failed to collect scheduled query profile for %s: %v", res.Name, err)
		} else {
			bt.pub.PublishQueryProfile(config.QueryProfileDatastream(ns), res.Name, "", responseID, profile, nil)
		}
	}

	bt.pub.PublishScheduledResponse(scheduleID, qi.PackID, qi.SpaceID, responseID, runTime, runTime, plannedScheduleTime, totalHits, scheduleExecutionCount)
}

func queryResultMeta(typ, action string, res QueryResult, scheduleExecutionCount int64, plannedScheduleTime time.Time) map[string]interface{} {
	m := map[string]interface{}{
		"type":                     typ,
		"calendar_type":            res.CalendarTime,
		"unix_time":                res.UnixTime,
		"planned_schedule_time":    plannedScheduleTime.Format(time.RFC3339Nano),
		"epoch":                    res.Epoch,
		"counter":                  res.Counter,
		"schedule_execution_count": scheduleExecutionCount,
	}

	if action != "" {
		m["action"] = action
	}
	return m
}

func (bt *osquerybeat) setManagerPayload(b *beat.Beat) {
	if b.Manager != nil {
		b.Manager.SetPayload(map[string]interface{}{
			"osquery_version": bt.osqueryVersion,
			"osquery_source":  bt.osquerySource,
		})
	}
}

type osqueryRuntimeSelection struct {
	BinDir        string
	ExtensionPath string
	Version       string
	Source        string
}

func (bt *osquerybeat) resolveOsqueryRuntime(ctx context.Context) (osqueryRuntimeSelection, error) {
	execPathFn := bt.executablePath
	if execPathFn == nil {
		execPathFn = os.Executable
	}
	exePath, err := execPathFn()
	if err != nil {
		return osqueryRuntimeSelection{}, err
	}
	bundledDir := filepath.Dir(exePath)

	bundledVersion, err := install.VerifyOsqueryBinary(runtime.GOOS, bundledDir, bt.log)
	if err != nil {
		bt.log.Warnf("failed to validate bundled osquery binary, fallback to distro version metadata: %v", err)
		bundledVersion = distro.OsquerydVersion()
	}
	result := osqueryRuntimeSelection{
		Version: bundledVersion,
		Source:  "bundled",
	}

	installDir := bundledDir
	installCfg := bt.osqueryInstallConfig
	if !installCfg.EnabledForPlatform(runtime.GOOS, runtime.GOARCH) {
		if err := installartifact.RemoveInstalled(installDir); err != nil {
			bt.log.Warnf("failed to cleanup previous custom osquery install, continue with bundled osquery: %v", err)
		}
		return result, nil
	}

	installed, err := installartifact.Ensure(ctx, installCfg, installDir, bt.log)
	if err != nil {
		return osqueryRuntimeSelection{}, err
	}
	bundledExtPath := osqd.OsqueryExtensionPathForPlatform(runtime.GOOS, bundledDir)
	if _, err := os.Stat(bundledExtPath); err != nil {
		return osqueryRuntimeSelection{}, fmt.Errorf("bundled osquery extension is required for custom runtime: %w", err)
	}

	return osqueryRuntimeSelection{
		BinDir:        installed.BinDir,
		ExtensionPath: bundledExtPath,
		Version:       installed.Version,
		Source:        "custom_artifact",
	}, nil
}

// Stop stops osquerybeat.
func (bt *osquerybeat) Stop() {
	bt.close()
}

func (bt *osquerybeat) registerActionHandler(b *beat.Beat, cli *osqdcli.Client, configPlugin *ConfigPlugin, rah *resetableActionHandler) {
	if b.Manager == nil {
		return
	}

	ah := &actionHandler{
		log:       bt.log,
		inputType: osqueryInputType,
		publisher: bt.pub,
		queryExec: cli,
		np:        configPlugin,
		profiles:  bt.liveProfiles,
	}
	rah.Attach(ah)
	b.Manager.RegisterAction(rah)
}

func (bt *osquerybeat) unregisterActionHandler(b *beat.Beat, rah *resetableActionHandler) {
	if b.Manager != nil && rah != nil {
		b.Manager.UnregisterAction(rah)
	}
}
