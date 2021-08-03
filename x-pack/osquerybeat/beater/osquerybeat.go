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
	"github.com/kolide/osquery-go"
	kconfig "github.com/kolide/osquery-go/plugin/config"
	klogger "github.com/kolide/osquery-go/plugin/logger"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
)

var (
	ErrInvalidQueryConfig = errors.New("invalid query configuration")
	ErrAlreadyRunning     = errors.New("already running")
	ErrQueryExecution     = errors.New("failed query execution")
	ErrActionRequest      = errors.New("invalid action request")
)

const (
	scheduledOsqueriesTypesCacheSize = 256 // Default number of queries types kept in memory to avoid fetching GetQueryColumns all the time
	adhocOsqueriesTypesCacheSize     = 256 // The final cache size equals the number of periodic queries plus this value, in order to have additional cache for ad-hoc queries

	limitQueryAtTime = 1 // Always run only one osquery query at a time. Addresses the issue: https://github.com/elastic/beats/issues/25297

	// The interval in second for configuration refresh;
	// osqueryd child process requests configuration from the configuration plugin implemented in osquerybeat
	configurationRefreshIntervalSecs = 60

	osqueryTimeout = 60 * time.Second
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
	client beat.Client

	log *logp.Logger

	// Beat lifecycle context, cancelled on Stop
	cancel context.CancelFunc
	mx     sync.Mutex
}

// New creates an instance of osquerybeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	log := logp.NewLogger("osquerybeat")

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &osquerybeat{
		b:      b,
		config: c,
		log:    log,
	}

	return bt, nil
}

func (bt *osquerybeat) initContext() (context.Context, error) {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.cancel != nil {
		return nil, ErrAlreadyRunning
	}
	var ctx context.Context
	ctx, bt.cancel = context.WithCancel(context.Background())
	return ctx, nil
}

func (bt *osquerybeat) close() {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.client != nil {
		bt.client.Close()
		bt.client = nil
	}
	if bt.cancel != nil {
		bt.cancel()
		bt.cancel = nil
	}
}

// Run starts osquerybeat.
func (bt *osquerybeat) Run(b *beat.Beat) error {
	ctx, err := bt.initContext()
	if err != nil {
		return err
	}
	defer bt.close()

	// Watch input configuration updates
	inputConfigCh := config.WatchInputs(ctx)

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
	osq := osqd.New(
		socketPath,
		osqd.WithLogger(bt.log),
		osqd.WithConfigRefresh(configurationRefreshIntervalSecs),
		osqd.WithConfigPlugin(configPluginName),
		osqd.WithLoggerPlugin(loggerPluginName),
	)

	// Check that osqueryd exists and runnable
	err = osq.Check(ctx)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	var inputCh chan []config.InputConfig

	startOsqueryIfNotStarted := func() {
		// Start only once
		if inputCh == nil {
			inputCh = make(chan []config.InputConfig, 1)
			g.Go(func() error {
				err := bt.runOsquery(ctx, b, osq, inputCh)
				if err != nil {
					bt.log.Errorf("Failed to run osqueryd: %v", err)
				}
				return err
			})
		}
	}

	// Start osquery only if config has inputs, otherwise it will be started on the first configuration sent from the agent
	// This way we don't need to persist the configuration for configuration plugin, because osquery is not running until
	// we have the first valid configuration
	if len(bt.config.Inputs) > 0 {
		startOsqueryIfNotStarted()
		inputCh <- bt.config.Inputs
	}

	// Set the osquery beat version to the manager payload. This allows the bundled osquery version to be reported to the stack.
	bt.setManagerPayload(b)

	// Run main loop
	g.Go(func() error {
		// Set initial queries from beats config if defined
		var processors *processors.Processors
		if len(bt.config.Inputs) > 0 {
			// Connect publisher
			processors, err = bt.reconnectPublisher(b, bt.config.Inputs)
			if err != nil {
				return err
			}
		}

		for {
			select {
			case <-ctx.Done():
				bt.log.Info("context cancelled, exiting")
				return ctx.Err()
			case inputConfigs := <-inputConfigCh:
				// Only set processor if it was not set before
				// TODO: implement a proper input/streams/processors manager, one publisher per input stream
				if processors == nil {
					processors, err = bt.reconnectPublisher(b, inputConfigs)
					if err != nil {
						bt.log.Errorf("Failed to connect beat publisher client, err: %v", err)
						return err
					}
				}
				startOsqueryIfNotStarted()
				inputCh <- inputConfigs
			}
		}
	})

	// Wait for clean exit
	return g.Wait()
}

func (bt *osquerybeat) runOsquery(ctx context.Context, b *beat.Beat, osq *osqd.OSQueryD, inputCh <-chan []config.InputConfig) error {
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
		err := osq.Run(ctx)
		if err != nil {
			bt.log.Errorf("Failed to run osqueryd: %v", err)
		}
		return err
	})

	// Create osqueryd client
	cli := osqdcli.New(socketPath,
		osqdcli.WithLogger(bt.log),
		osqdcli.WithTimeout(osqueryTimeout),
		osqdcli.WithCache(cache, adhocOsqueriesTypesCacheSize),
	)

	// Create osquery configuration plugin that loads a persisted configuration from the disk
	configPlugin := NewConfigPlugin(bt.log)
	// Resize cache
	cache.Resize(configPlugin.Count())

	// Create osquery logger plugin
	loggerPlugin := NewLoggerPlugin(bt.log, func(res SnapshotResult) {
		bt.handleSnapshotResult(ctx, cli, configPlugin, res)
	})

	// Run extensions
	g.Go(func() error {
		return runExtensionServer(ctx, socketPath, configPlugin, loggerPlugin, osqueryTimeout)
	})

	// Run main loop
	g.Go(func() error {
		// Connect to osqueryd
		err = cli.Connect(ctx)
		if err != nil {
			return err
		}
		defer cli.Close()

		// Register action handler
		ah := bt.registerActionHandler(b, cli)
		defer bt.unregisterActionHandler(b, ah)

		// Process input
		for {
			select {
			case <-ctx.Done():
				bt.log.Info("context cancelled, exiting")
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
	return g.Wait()
}

func runExtensionServer(ctx context.Context, socketPath string, configPlugin *ConfigPlugin, loggerPlugin *LoggerPlugin, timeout time.Duration) (err error) {
	// Register config and logger extensions
	extserver, err := osquery.NewExtensionManagerServer(extManagerServerName, socketPath, osquery.ServerTimeout(timeout))
	if err != nil {
		return
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
		for {
			select {
			case <-ctx.Done():
				return extserver.Shutdown(context.Background())
			}
		}
	})

	return g.Wait()
}

func (bt *osquerybeat) handleSnapshotResult(ctx context.Context, cli *osqdcli.Client, configPlugin *ConfigPlugin, res SnapshotResult) {
	sql, ok := configPlugin.ResolveName(res.Name)
	if !ok {
		bt.log.Errorf("failed to resolve query name: %s", res.Name)
		return
	}

	hits, err := cli.ResolveResult(ctx, sql, res.Hits)
	if err != nil {
		bt.log.Errorf("failed to resolve query types: %s", res.Name)
		return
	}

	// Map to ECS
	var ecsFields []common.MapStr
	mapping, ok := configPlugin.LookupECSMapping(res.Name)
	if ok && len(mapping) > 0 {
		ecsFields = make([]common.MapStr, len(hits))
		for i, hit := range hits {
			ecsFields[i] = common.MapStr(mapping.Map(hit))
		}
	} else {
		// ECS mapping is optional, continue
		bt.log.Debugf("ECS mapping is not found for query name: %s", res.Name)
	}

	responseID := uuid.Must(uuid.NewV4()).String()
	bt.publishEvents(config.DefaultStreamIndex, res.Name, responseID, hits, ecsFields, nil)
}

func (bt *osquerybeat) setManagerPayload(b *beat.Beat) {
	if b.Manager != nil {
		b.Manager.SetPayload(map[string]interface{}{
			"osquery_version": distro.OsquerydVersion(),
		})
	}
}

func (bt *osquerybeat) reconnectPublisher(b *beat.Beat, inputs []config.InputConfig) (*processors.Processors, error) {
	processors, err := processorsForInputsConfig(inputs)
	if err != nil {
		return nil, err
	}

	bt.log.Debugf("Connect publisher with processors: %d", len(processors.All()))
	// Connect publisher
	client, err := b.Publisher.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			Processor: processors,
		},
	})
	if err != nil {
		return nil, err
	}

	// Swap client
	bt.mx.Lock()
	defer bt.mx.Unlock()
	oldclient := bt.client
	bt.client = client
	if oldclient != nil {
		oldclient.Close()
	}
	return processors, nil
}

func processorsForInputsConfig(inputs []config.InputConfig) (procs *processors.Processors, err error) {
	// Use only first input processor
	// Every input will have a processor that adds the elastic_agent info, we need only one
	// Not expecting other processors at the moment and this needs to work for 7.13
	for _, input := range inputs {
		if len(input.Processors) > 0 {
			procs, err = processors.New(input.Processors)
			if err != nil {
				return nil, err
			}
			return procs, nil
		}
	}
	return nil, nil
}

// Stop stops osquerybeat.
func (bt *osquerybeat) Stop() {
	bt.close()
}

func (bt *osquerybeat) registerActionHandler(b *beat.Beat, cli *osqdcli.Client) *actionHandler {
	if b.Manager == nil {
		return nil
	}

	ah := &actionHandler{
		log:       bt.log,
		inputType: osqueryInputType,
		bt:        bt,
		cli:       cli,
	}
	b.Manager.RegisterAction(ah)
	return ah
}

func (bt *osquerybeat) unregisterActionHandler(b *beat.Beat, ah *actionHandler) {
	if b.Manager != nil && ah != nil {
		b.Manager.UnregisterAction(ah)
	}
}

func (bt *osquerybeat) publishEvents(index, actionID, responseID string, hits []map[string]interface{}, ecsFields []common.MapStr, reqData interface{}) {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	for i, hit := range hits {
		var fields common.MapStr

		if len(ecsFields) > i {
			fields = ecsFields[i]
		} else {
			fields = common.MapStr{}
		}

		fields["type"] = bt.b.Info.Name
		fields["action_id"] = actionID
		fields["osquery"] = hit

		event := beat.Event{
			Timestamp: time.Now(),
			Fields:    fields,
		}

		if reqData != nil {
			event.Fields["action_data"] = reqData
		}

		if responseID != "" {
			event.Fields["response_id"] = responseID
		}
		if index != "" {
			event.Meta = common.MapStr{events.FieldMetaRawIndex: index}
		}

		bt.client.Publish(event)
	}
	bt.log.Infof("%d events sent to index %s", len(hits), index)
}
