// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/osquery/osquery-go"
	kconfig "github.com/osquery/osquery-go/plugin/config"
	klogger "github.com/osquery/osquery-go/plugin/logger"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/proc"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
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
	osqueryMaxTimeout = 15 * time.Minute
)

const (
	osqueryInputType     = "osquery"
	extManagerServerName = "osqextman"
	configPluginName     = "osq_config"
	loggerPluginName     = "osq_logger"
)

// osquerybeat configuration.
type osquerybeat struct {
	b      *beat.Beat
	config config.Config

	pub *pub.Publisher

	log *logp.Logger

	// Beat lifecycle context, cancelled on Stop
	cancel context.CancelFunc
	mx     sync.Mutex

	// parent process watcher
	watcher *Watcher
}

// New creates an instance of osquerybeat.
func New(b *beat.Beat, cfg *conf.C) (beat.Beater, error) {
	log := logp.NewLogger("osquerybeat")

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	bt := &osquerybeat{
		b:      b,
		config: c,
		log:    log,
		pub:    pub.New(b, log),
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
	inputConfigCh := config.WatchInputs(ctx, bt.log)

	// Install osqueryd if needed
	err = installOsquery(ctx)
	if err != nil {
		return err
	}

	// Create socket path
	socketPath, cleanupFn, err := osqd.CreateSocketPath()
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Create osqueryd runner
	osq, err := osqd.New(
		socketPath,
		osqd.WithLogger(bt.log),
		osqd.WithConfigRefresh(configurationRefreshIntervalSecs),
		osqd.WithConfigPlugin(configPluginName),
		osqd.WithLoggerPlugin(loggerPluginName),
	)

	if err != nil {
		return err
	}

	// Check that osqueryd exists and runnable
	err = osq.Check(ctx)
	if err != nil {
		return err
	}

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
			return bt.runOsquery(ctx, b, osq, flags, inputCh, rah)
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
	if err := b.Manager.Start(); err != nil {
		return err
	}
	defer b.Manager.Stop()

	// Set the osquery beat version to the manager payload. This allows the bundled osquery version to be reported to the stack.
	bt.setManagerPayload(b)

	// Run main loop
	g.Go(func() error {
		// Configure publisher from initial input
		err := bt.pub.Configure(bt.config.Inputs)
		if err != nil {
			return err
		}

		for {
			select {
			case <-ctx.Done():
				bt.log.Info("osquerybeat context cancelled, exiting")
				return ctx.Err()
			case inputConfigs := <-inputConfigCh:
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
			bt.log.Debugf("osquerybeat Run exited, context cancelled")
		} else {
			bt.log.Errorf("osquerybeat Run exited with error: %v", err)
		}
	} else {
		bt.log.Debugf("osquerybeat Run exited")
	}
	return err
}

func (bt *osquerybeat) runOsquery(ctx context.Context, b *beat.Beat, osq *osqd.OSQueryD, flags osqd.Flags, inputCh <-chan []config.InputConfig, rah *resetableActionHandler) error {
	socketPath := osq.SocketPath()

	// Create a cache for queries types resolution
	cache, err := lru.New(adhocOsqueriesTypesCacheSize)
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
		defer cli.Close()

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
		bt.log.Errorf("runOsquery exited with error: %v", err)
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

	var (
		hits []map[string]interface{}
	)

	responseID := uuid.Must(uuid.NewV4()).String()

	if res.Action == "snapshot" {
		snapshot, err := cli.ResolveResult(ctx, qi.Query, res.Hits)
		if err != nil {
			bt.log.Errorf("failed to resolve snapshot query result types: %s", res.Name)
			return
		}
		hits = append(hits, snapshot...)
		meta := queryResultMeta("snapshot", "", res)
		bt.pub.Publish(config.Datastream(ns), res.Name, responseID, meta, hits, qi.ECSMapping, nil)
	} else {
		if len(res.DiffResults.Added) > 0 {
			added, err := cli.ResolveResult(ctx, qi.Query, res.DiffResults.Added)
			if err != nil {
				bt.log.Errorf(`failed to resolve diff query "added" result types: %s`, res.Name)
				return
			}
			hits = append(hits, added...)
			meta := queryResultMeta("diff", "added", res)
			bt.pub.Publish(config.Datastream(ns), res.Name, responseID, meta, hits, qi.ECSMapping, nil)
		}
		if len(res.DiffResults.Removed) > 0 {
			removed, err := cli.ResolveResult(ctx, qi.Query, res.DiffResults.Added)
			if err != nil {
				bt.log.Errorf(`failed to resolve diff query "removed" result types: %s`, res.Name)
				return
			}
			hits = append(hits, removed...)
			meta := queryResultMeta("diff", "removed", res)
			bt.pub.Publish(config.Datastream(ns), res.Name, responseID, meta, hits, qi.ECSMapping, nil)
		}
	}

}

func queryResultMeta(typ, action string, res QueryResult) map[string]interface{} {
	m := map[string]interface{}{
		"type":          typ,
		"calendar_type": res.CalendarTime,
		"unix_time":     res.UnixTime,
		"epoch":         res.Epoch,
		"counter":       res.Counter,
	}

	if action != "" {
		m["action"] = action
	}
	return m
}

func (bt *osquerybeat) setManagerPayload(b *beat.Beat) {
	if b.Manager != nil {
		b.Manager.SetPayload(map[string]interface{}{
			"osquery_version": distro.OsquerydVersion(),
		})
	}
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
	}
	rah.Attach(ah)
	b.Manager.RegisterAction(rah)
}

func (bt *osquerybeat) unregisterActionHandler(b *beat.Beat, rah *resetableActionHandler) {
	if b.Manager != nil && rah != nil {
		b.Manager.UnregisterAction(rah)
	}
}
