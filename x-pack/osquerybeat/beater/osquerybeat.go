// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

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

	var wg sync.WaitGroup

	exefp, err := os.Executable()
	if err != nil {
		return err
	}
	exedir := filepath.Dir(exefp)

	// Create temp directory for socket and possibly other things
	// The unix domain socker path is limited to 108 chars and would
	// not always be able to create in subdirectory
	tmpdir, removeTmpDir, err := createTempDir()
	if err != nil {
		return err
	}
	defer removeTmpDir()

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
	bt.client, err = b.Publisher.Connect()
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

	// Connect to osqueryd socket. Replying on the client library retry logic that checks for the socket availability
	bt.osqCli, err = osqueryd.NewClient(ctx, osd.SocketPath, osqueryd.DefaultTimeout)
	if err != nil {
		bt.log.Errorf("Failed to create osqueryd client: %v", err)
		return err
	}

	// Watch input configuration updates
	inputConfigCh := config.WatchInputs(ctx)

	// Start queries execution scheduler
	scheduler := NewScheduler(ctx, bt.query)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.Run()
	}()

	// Load initial queries
	streams, inputTypes := config.StreamsFromInputs(bt.config.Inputs)
	sz := len(streams)
	if sz > 0 {
		scheduler.Load(streams)
	}

	// Agent actions handlers
	var actionHandlers []*actionHandler
	unregisterActionHandlers := func() {
		// Unregister action handlers
		if b.Manager != nil {
			for _, ah := range actionHandlers {
				b.Manager.UnregisterAction(ah)
				ah.bt = nil
			}
		}
	}

	registerActionHandlers := func(itypes []string) {
		unregisterActionHandlers()
		// Register action handler
		if b.Manager != nil {
			for _, inType := range itypes {
				ah := &actionHandler{
					inputType: inType,
					bt:        bt,
				}
				b.Manager.RegisterAction(ah)
			}
		}
	}

	setManagerPayload := func(itypes []string) {
		if b.Manager != nil {
			b.Manager.SetPayload(map[string]interface{}{
				"osquery_version": distro.OsquerydVersion(),
			})
		}
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
			streams, inputTypes = config.StreamsFromInputs(inputConfigs)
			registerActionHandlers(inputTypes)
			setManagerPayload(inputTypes)
			scheduler.Load(streams)
		}
	}

	// Unregister action handlers
	unregisterActionHandlers()

	// Wait for clean scheduler exit
	wg.Wait()

	return err
}

// Stop stops osquerybeat.
func (bt *osquerybeat) Stop() {
	bt.close()
}

func createTempDir() (string, func(), error) {
	tpath, err := ioutil.TempDir("", "")
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return tpath, func() {
		os.RemoveAll(tpath)
	}, nil
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

func (bt *osquerybeat) executeQuery(ctx context.Context, log *logp.Logger, index, id, query, responseID string, req map[string]interface{}) error {
	log.Debugf("Execute query: %s", query)

	start := time.Now()

	hits, err := bt.osqCli.Query(ctx, query)

	if err != nil {
		log.Errorf("Failed to execute query, err: %v", err)
		return err
	}

	log.Infof("Completed query in: %v", time.Since(start))

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

	start := time.Now().UTC().Format(time.RFC3339Nano)
	err := a.execute(ctx, req)
	end := time.Now().UTC().Format(time.RFC3339Nano)

	res := map[string]interface{}{
		"started_at":   start,
		"completed_at": end,
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
