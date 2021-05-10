// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sync/semaphore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqueryd"
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
)

// osquerybeat configuration.
type osquerybeat struct {
	b      *beat.Beat
	config config.Config
	client beat.Client
	osqCli *osqueryd.Client

	log *logp.Logger

	// Beat lifecycle context, cancelled on Stop
	cancel context.CancelFunc
	mx     sync.Mutex

	// limiter to run one query at a time
	limitSem *semaphore.Weighted
}

// New creates an instance of osquerybeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	log := logp.NewLogger("osquerybeat")

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &osquerybeat{
		b:        b,
		config:   c,
		log:      log,
		limitSem: semaphore.NewWeighted(limitQueryAtTime),
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

func (bt *osquerybeat) inputTypes() []string {
	m := make(map[string]struct{})
	for _, input := range bt.config.Inputs {
		m[input.Type] = struct{}{}
	}

	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}

	return res
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

	var wg sync.WaitGroup

	exefp, err := os.Executable()
	if err != nil {
		return err
	}
	exedir := filepath.Dir(exefp)

	// Create temp directory for socket and possibly other things
	// The unix domain socker path is limited to 108 chars and would
	// not always be able to create in subdirectory
	tmpdir, removeTmpDir, err := createSockDir(bt.log)
	if err != nil {
		return err
	}
	defer func() {
		if removeTmpDir != nil {
			removeTmpDir()
		}
	}()

	// Install osqueryd if needed
	err = installOsquery(ctx, exedir)
	if err != nil {
		return err
	}

	// Start osqueryd child process
	osd := osqueryd.OsqueryD{
		RootDir:    exedir,
		SocketPath: osqueryd.SocketPath(tmpdir),
	}

	// Connect publisher
	processors, err := bt.reconnectPublisher(b, bt.config.Inputs)
	if err != nil {
		return err
	}

	// Start osqueryd child process
	osdCtx, osdCn := context.WithCancel(ctx)
	defer osdCn()
	osqDone, err := osd.Start(osdCtx)
	if err != nil {
		bt.log.Errorf("Failed to start osqueryd process: %v", err)
		return err
	}

	// Create a cache for queries
	cache, err := lru.New(scheduledOsqueriesTypesCacheSize + adhocOsqueriesTypesCacheSize)
	if err != nil {
		bt.log.Errorf("Failed to create osquery query results types cache: %v", err)
		return err
	}

	// Connect to osqueryd socket. Replying on the client library retry logic that checks for the socket availability
	bt.osqCli, err = osqueryd.NewClient(ctx, osd.SocketPath, osqueryd.DefaultTimeout, bt.log, osqueryd.WithCache(cache))
	if err != nil {
		bt.log.Errorf("Failed to create osqueryd client: %v", err)
		return err
	}

	cacheResize := func(size int) {
		if size <= 0 {
			size = scheduledOsqueriesTypesCacheSize
		}
		cache.Resize(size + adhocOsqueriesTypesCacheSize)
	}

	// Unlink socket path early
	if removeTmpDir != nil {
		removeTmpDir()
		removeTmpDir = nil
	}

	// Start queries execution scheduler
	schedCtx, schedCancel := context.WithCancel(ctx)
	scheduler := NewScheduler(schedCtx, bt.query)
	defer schedCancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.Run()
	}()

	// Load initial queries
	loadSchedulerStreams := func(streams []config.StreamConfig) {
		cacheResize(len(streams))
		scheduler.Load(streams)
	}
	streams, inputTypes := config.StreamsFromInputs(bt.config.Inputs)
	sz := len(streams)
	if sz > 0 {
		loadSchedulerStreams(streams)
	}

	// Agent actions handlers
	var actionHandlers []*actionHandler
	unregisterActionHandlers := func() {
		bt.log.Debug("unregisterActionHandlers")
		// Unregister action handlers
		if b.Manager != nil {
			for _, ah := range actionHandlers {
				b.Manager.UnregisterAction(ah)
				ah.bt = nil
			}
		}
		actionHandlers = nil
	}

	registerActionHandlers := func(itypes []string) {
		unregisterActionHandlers()
		// Register action handler
		if b.Manager != nil {
			bt.log.Debugf("registerActionHandlers register actions: %v", itypes)
			for _, inType := range itypes {
				ah := &actionHandler{
					inputType: inType,
					bt:        bt,
				}
				b.Manager.RegisterAction(ah)
				actionHandlers = append(actionHandlers, ah)
			}
		} else {
			bt.log.Debug("registerActionHandlers b.Manager is nil, not registering actions")
		}
	}

	handleInputConfig := func(inputConfigs []config.InputConfig) error {
		bt.log.Debug("Handle input configuration change")
		// Only set processor if it was not set before
		if processors == nil {
			bt.log.Debug("Set processors for the first time")
			var err error
			processors, err = bt.reconnectPublisher(b, inputConfigs)
			if err != nil {
				bt.log.Errorf("Failed to connect beat publisher client, err: %v", err)
				return err
			}
		} else {
			bt.log.Debug("Processors are already set")
		}

		streams, inputTypes = config.StreamsFromInputs(inputConfigs)
		registerActionHandlers(inputTypes)
		bt.setManagerPayload(b)
		loadSchedulerStreams(streams)
		return nil
	}

LOOP:
	for {
		select {
		case err = <-osqDone:
			break LOOP // Exiting if osquery child process exited with error
		case <-ctx.Done():
			bt.log.Info("Wait osqueryd exit")
			exitErr := <-osqDone
			bt.log.Infof("Exited osqueryd process, error: %v", exitErr)
			break LOOP
		case inputConfigs := <-inputConfigCh:
			err = handleInputConfig(inputConfigs)
			if err != nil {
				bt.log.Errorf("Failed to handle input configuration, err: %v, exiting", err)
				// Cancel scheduler
				schedCancel()
				break LOOP
			}
		}
	}

	// Unregister action handlers
	unregisterActionHandlers()

	// Wait for clean scheduler exit
	bt.log.Debug("Wait clean beat run exit")
	wg.Wait()
	bt.log.Debug("Beat run exited, err: %v", err)

	return err
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

func (bt *osquerybeat) query(ctx context.Context, q interface{}) error {
	cfg, ok := q.(config.StreamConfig)
	if !ok {
		bt.log.Error("Unexpected query configuration")
		return ErrInvalidQueryConfig
	}

	// Response ID could be useful in order to differentiate between different runs for the interval queries
	responseID := uuid.Must(uuid.NewV4()).String()

	log := bt.log.With("id", cfg.ID).With("query", cfg.Query).With("interval", cfg.Interval)

	reqData := map[string]interface{}{
		"id":    cfg.ID,
		"query": cfg.Query,
	}

	err := bt.executeQuery(ctx, log, cfg.Index, cfg.ID, cfg.Query, responseID, reqData)
	if err != nil {
		// Preserving the error as is, it will be attached to the result document
		return err
	}
	return nil
}

func (bt *osquerybeat) executeQueryWithLimiter(ctx context.Context, log *logp.Logger, query string) ([]map[string]interface{}, error) {
	// This limits the execution of query to one at a time.
	// Concurrent use of osqueryd socket lead to failures/errors.
	// Example: osquery failed: *osquery.ExtensionResponse error reading struct: error reading field 0: read unix ->/var/run/404419649/osquery.sock: i/o timeout"
	// The scheduled and ad-hoc queries use the same code path at the moment.
	// The plan for the next release is to switch the scheduled queries to use osqueryd scheduler instead.
	err := bt.limitSem.Acquire(ctx, limitQueryAtTime)
	if err != nil {
		return nil, err
	}
	defer bt.limitSem.Release(limitQueryAtTime)

	// "If ctx is already done, Acquire may still succeed without blocking."
	// https://github.com/golang/sync/blob/master/semaphore/semaphore.go#L68
	if ctx.Err() != nil {
		return nil, err
	}

	log.Debugf("Execute query: %s", query)

	start := time.Now()

	hits, err := bt.osqCli.Query(ctx, query)

	if err != nil {
		log.Errorf("Failed to execute query, err: %v", err)
		return nil, err
	}

	log.Infof("Completed query in: %v", time.Since(start))
	return hits, nil
}

func (bt *osquerybeat) executeQuery(ctx context.Context, log *logp.Logger, index, id, query, responseID string, req map[string]interface{}) error {

	hits, err := bt.executeQueryWithLimiter(ctx, log, query)
	if err != nil {
		return err
	}

	bt.mx.Lock()
	defer bt.mx.Unlock()
	for _, hit := range hits {
		reqData := req["data"]
		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":      bt.b.Info.Name,
				"action_id": id,
				"osquery":   hit,
			},
		}
		if reqData != nil {
			event.Fields["action_data"] = reqData
		}
		if responseID != "" {
			event.Fields["response_id"] = responseID
		}
		if index != "" {
			event.Meta = common.MapStr{"index": index}
		}

		bt.client.Publish(event)
	}
	log.Infof("The %d events sent to index %s", len(hits), index)
	return nil
}

type actionHandler struct {
	inputType string
	bt        *osquerybeat
}

func (a *actionHandler) Name() string {
	return a.inputType
}

type actionData struct {
	Query string
	ID    string
}

func actionDataFromRequest(req map[string]interface{}) (ad actionData, err error) {
	if req == nil {
		return ad, ErrActionRequest
	}
	if v, ok := req["id"]; ok {
		if id, ok := v.(string); ok {
			ad.ID = id
		}
	}
	if v, ok := req["data"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			if v, ok := m["query"]; ok {
				if query, ok := v.(string); ok {
					ad.Query = query
				}
			}
		}
	}
	return ad, nil
}

// Execute handles the action request.
func (a *actionHandler) Execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {

	start := time.Now().UTC()
	err := a.execute(ctx, req)
	end := time.Now().UTC()

	res := map[string]interface{}{
		"started_at":   start.Format(time.RFC3339Nano),
		"completed_at": end.Format(time.RFC3339Nano),
	}

	if err != nil {
		res["error"] = err.Error()
	}
	return res, nil
}

func (a *actionHandler) execute(ctx context.Context, req map[string]interface{}) error {
	ad, err := actionDataFromRequest(req)
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrQueryExecution)
	}
	return a.bt.executeQuery(ctx, a.bt.log, config.DefaultStreamIndex, ad.ID, ad.Query, "", req)
}
