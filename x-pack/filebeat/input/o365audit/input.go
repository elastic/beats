// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/o365audit/poll"
)

const (
	inputName    = "o365audit"
	fieldsPrefix = inputName
)

func init() {
	if err := input.Register(inputName, NewInput); err != nil {
		panic(errors.Wrapf(err, "unable to create %s input", inputName))
	}
}

type o365input struct {
	config  Config
	outlet  channel.Outleter
	storage *stateStorage
	log     *logp.Logger
	pollers map[stream]*poll.Poller
	cancel  func()
	ctx     context.Context
	wg      sync.WaitGroup
	runOnce sync.Once
}

type apiEnvironment struct {
	TenantID    string
	ContentType string
	Config      APIConfig
	Callback    func(beat.Event) bool
	Logger      *logp.Logger
	Clock       func() time.Time
}

// NewInput creates a new o365audit input.
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	cfgwarn.Beta("The %s input is beta", inputName)
	inp, err = newInput(cfg, connector, inputContext)
	return inp, errors.Wrap(err, inputName)
}

func newInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading config")
	}

	log := logp.NewLogger(inputName)

	// TODO: Update with input v2 state.
	storage := newStateStorage(noopPersister{})

	var out channel.Outleter
	out, err = connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
		ACKLastEvent: func(private interface{}) {
			// Errors don't have a cursor.
			if cursor, ok := private.(cursor); ok {
				log.Debugf("ACKed cursor %+v", cursor)
				if err := storage.Save(cursor); err != nil && err != errNoUpdate {
					log.Errorf("Error saving state: %v", err)
				}
			}
		},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	pollers := make(map[stream]*poll.Poller)
	for _, tenantID := range config.TenantID {
		// MaxRequestsPerMinute limitation is per tenant.
		delay := time.Duration(len(config.ContentType)) * time.Minute / time.Duration(config.API.MaxRequestsPerMinute)
		auth, err := config.NewTokenProvider(tenantID)
		if err != nil {
			return nil, err
		}
		if _, err = auth.Token(); err != nil {
			return nil, errors.Wrapf(err, "unable to acquire authentication token for tenant:%s", tenantID)
		}
		for _, contentType := range config.ContentType {
			key := stream{
				tenantID:    tenantID,
				contentType: contentType,
			}
			poller, err := poll.New(
				poll.WithTokenProvider(auth),
				poll.WithMinRequestInterval(delay),
				poll.WithLogger(log.With("tenantID", tenantID, "contentType", contentType)),
				poll.WithContext(ctx),
				poll.WithRequestDecorator(
					autorest.WithUserAgent(useragent.UserAgent("Filebeat-"+inputName)),
					autorest.WithQueryParameters(common.MapStr{
						"publisherIdentifier": tenantID,
					}),
				),
			)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create API poller")
			}
			pollers[key] = poller
		}
	}

	return &o365input{
		config:  config,
		outlet:  out,
		storage: storage,
		log:     log,
		pollers: pollers,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Run starts the o365input. Only has effect the first time it's called.
func (inp *o365input) Run() {
	inp.runOnce.Do(inp.run)
}

func (inp *o365input) run() {
	for stream, poller := range inp.pollers {
		start := inp.loadLastLocation(stream)
		inp.log.Infow("Start fetching events",
			"cursor", start,
			"tenantID", stream.tenantID,
			"contentType", stream.contentType)
		inp.runPoller(poller, start)
	}
}

func (inp *o365input) runPoller(poller *poll.Poller, start cursor) {
	ctx := apiEnvironment{
		TenantID:    start.tenantID,
		ContentType: start.contentType,
		Config:      inp.config.API,
		Callback:    inp.reportEvent,
		Logger:      poller.Logger(),
		Clock:       time.Now,
	}
	inp.wg.Add(1)
	go func() {
		defer logp.Recover("panic in " + inputName + " runner.")
		defer inp.wg.Done()
		action := ListBlob(start, ctx)
		// When resuming from a saved state, it's necessary to query for the
		// same startTime that provided the last ACKed event. Otherwise there's
		// the risk of observing partial blobs with different line counts, due to
		// how the backend works.
		if start.line > 0 {
			action = action.WithStartTime(start.startTime)
		}
		if err := poller.Run(action); err != nil {
			ctx.Logger.Errorf("API polling terminated with error: %v", err.Error())
			msg := common.MapStr{}
			msg.Put("error.message", err.Error())
			msg.Put("event.kind", "pipeline_error")
			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    msg,
			}
			inp.reportEvent(event)
		}
	}()
}

func (inp *o365input) reportEvent(event beat.Event) bool {
	return inp.outlet.OnEvent(event)
}

// Stop terminates the o365 input.
func (inp *o365input) Stop() {
	inp.log.Info("Stopping input " + inputName)
	defer inp.log.Info(inputName + " stopped.")
	defer inp.outlet.Close()
	inp.cancel()
}

// Wait terminates the o365input and waits for all the pollers to finalize.
func (inp *o365input) Wait() {
	inp.Stop()
	inp.wg.Wait()
}

func (inp *o365input) loadLastLocation(key stream) cursor {
	period := inp.config.API.MaxRetention
	retentionLimit := time.Now().UTC().Add(-period)
	cursor, err := inp.storage.Load(key)
	if err != nil {
		if err == errStateNotFound {
			inp.log.Infof("No saved state found. Will fetch events for the last %v.", period.String())
		} else {
			inp.log.Errorw("Error loading saved state. Will fetch all retained events. "+
				"Depending on max_retention, this can cause event loss or duplication.",
				"error", err,
				"max_retention", period.String())
		}
		cursor.timestamp = retentionLimit
	}
	if cursor.timestamp.Before(retentionLimit) {
		inp.log.Warnw("Last update exceeds the retention limit. "+
			"Probably some events have been lost.",
			"resume_since", cursor,
			"retention_limit", retentionLimit,
			"max_retention", period.String())
		// Due to API limitations, it's necessary to perform a query for each
		// day. These avoids performing a lot of queries that will return empty
		// when the input hasn't run in a long time.
		cursor.timestamp = retentionLimit
	}
	return cursor
}

var errTerminated = errors.New("terminated due to output closed")

// Report returns an action that produces a beat.Event from the given object.
func (env apiEnvironment) Report(doc common.MapStr, private interface{}) poll.Action {
	return func(poll.Enqueuer) error {
		if !env.Callback(env.toBeatEvent(doc, private)) {
			return errTerminated
		}
		return nil
	}
}

// ReportAPIError returns an action that produces a beat.Event from an API error.
func (env apiEnvironment) ReportAPIError(err apiError) poll.Action {
	return func(poll.Enqueuer) error {
		if !env.Callback(err.ToBeatEvent()) {
			return errTerminated
		}
		return nil
	}
}

func (env apiEnvironment) toBeatEvent(doc common.MapStr, private interface{}) beat.Event {
	var errs multierror.Errors
	ts, err := getDateKey(doc, "CreationTime", apiDateFormats)
	if err != nil {
		ts = time.Now()
		errs = append(errs, errors.Wrap(err, "failed parsing CreationTime"))
	}
	b := beat.Event{
		Timestamp: ts,
		Fields: common.MapStr{
			fieldsPrefix: doc,
		},
		Private: private,
	}
	if env.Config.SetIDFromAuditRecord {
		if id, err := getString(doc, "Id"); err == nil && len(id) > 0 {
			b.SetID(id)
		}
	}
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for idx, e := range errs {
			msgs[idx] = e.Error()
		}
		b.PutValue("error.message", msgs)
	}
	return b
}
