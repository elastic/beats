// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	cursor "github.com/menderesk/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/useragent"
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/o365audit/poll"
	"github.com/menderesk/go-concert/ctxtool"
	"github.com/menderesk/go-concert/timed"
)

const (
	pluginName   = "o365audit"
	fieldsPrefix = pluginName
)

type o365input struct {
	config Config
}

// Stream represents an event stream.
type stream struct {
	tenantID    string
	contentType string
}

type apiEnvironment struct {
	TenantID    string
	ContentType string
	Config      APIConfig
	Callback    func(event beat.Event, cursor interface{}) error
	Logger      *logp.Logger
	Clock       func() time.Time
}

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "O365 logs",
		Doc:        "Collect logs from O365 service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

func configure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, errors.Wrap(err, "reading config")
	}

	var sources []cursor.Source
	for _, tenantID := range config.TenantID {
		for _, contentType := range config.ContentType {
			sources = append(sources, &stream{
				tenantID:    tenantID,
				contentType: contentType,
			})
		}
	}

	return sources, &o365input{config: config}, nil
}

func (s *stream) Name() string {
	return s.tenantID + "::" + s.contentType
}

func (inp *o365input) Name() string { return pluginName }

func (inp *o365input) Test(src cursor.Source, ctx v2.TestContext) error {
	tenantID := src.(*stream).tenantID
	auth, err := inp.config.NewTokenProvider(tenantID)
	if err != nil {
		return err
	}

	if _, err := auth.Token(); err != nil {
		return errors.Wrapf(err, "unable to acquire authentication token for tenant:%s", tenantID)
	}

	return nil
}

func (inp *o365input) Run(
	ctx v2.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	for ctx.Cancelation.Err() == nil {
		err := inp.runOnce(ctx, src, cursor, publisher)
		if err == nil {
			break
		}
		if ctx.Cancelation.Err() != err && err != context.Canceled {
			msg := common.MapStr{}
			msg.Put("error.message", err.Error())
			msg.Put("event.kind", "pipeline_error")
			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    msg,
			}
			publisher.Publish(event, nil)
			ctx.Logger.Errorf("Input failed: %v", err)
			ctx.Logger.Infof("Restarting in %v", inp.config.API.ErrorRetryInterval)
			timed.Wait(ctx.Cancelation, inp.config.API.ErrorRetryInterval)
		}
	}
	return nil
}

func (inp *o365input) runOnce(
	ctx v2.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	stream := src.(*stream)
	tenantID, contentType := stream.tenantID, stream.contentType
	log := ctx.Logger.With("tenantID", tenantID, "contentType", contentType)

	tokenProvider, err := inp.config.NewTokenProvider(stream.tenantID)
	if err != nil {
		return err
	}

	if _, err := tokenProvider.Token(); err != nil {
		return errors.Wrapf(err, "unable to acquire authentication token for tenant:%s", stream.tenantID)
	}

	config := &inp.config

	// MaxRequestsPerMinute limitation is per tenant.
	delay := time.Duration(len(config.ContentType)) * time.Minute / time.Duration(config.API.MaxRequestsPerMinute)

	poller, err := poll.New(
		poll.WithTokenProvider(tokenProvider),
		poll.WithMinRequestInterval(delay),
		poll.WithLogger(log),
		poll.WithContext(ctxtool.FromCanceller(ctx.Cancelation)),
		poll.WithRequestDecorator(
			autorest.WithUserAgent(useragent.UserAgent("Filebeat-"+pluginName)),
			autorest.WithQueryParameters(common.MapStr{
				"publisherIdentifier": tenantID,
			}),
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create API poller")
	}

	start := initCheckpoint(log, cursor, config.API.MaxRetention)
	action := makeListBlob(start, apiEnvironment{
		Logger:      log,
		TenantID:    tenantID,
		ContentType: contentType,
		Config:      inp.config.API,
		Callback:    publisher.Publish,
		Clock:       time.Now,
	})
	if start.Line > 0 {
		action = action.WithStartTime(start.StartTime)
	}

	log.Infow("Start fetching events", "cursor", start)
	return poller.Run(action)
}

func initCheckpoint(log *logp.Logger, c cursor.Cursor, maxRetention time.Duration) checkpoint {
	var cp checkpoint
	retentionLimit := time.Now().UTC().Add(-maxRetention)

	if c.IsNew() {
		log.Infof("No saved state found. Will fetch events for the last %v.", maxRetention.String())
		cp.Timestamp = retentionLimit
	} else {
		err := c.Unpack(&cp)
		if err != nil {
			log.Errorw("Error loading saved state. Will fetch all retained events. "+
				"Depending on max_retention, this can cause event loss or duplication.",
				"error", err,
				"max_retention", maxRetention.String())
			cp.Timestamp = retentionLimit
		}
	}

	if cp.Timestamp.Before(retentionLimit) {
		log.Warnw("Last update exceeds the retention limit. "+
			"Probably some events have been lost.",
			"resume_since", cp,
			"retention_limit", retentionLimit,
			"max_retention", maxRetention.String())
		// Due to API limitations, it's necessary to perform a query for each
		// day. These avoids performing a lot of queries that will return empty
		// when the input hasn't run in a long time.
		cp.Timestamp = retentionLimit
	}

	return cp
}

// Report returns an action that produces a beat.Event from the given object.
func (env apiEnvironment) Report(raw json.RawMessage, doc common.MapStr, private interface{}) poll.Action {
	return func(poll.Enqueuer) error {
		return env.Callback(env.toBeatEvent(raw, doc), private)
	}
}

// ReportAPIError returns an action that produces a beat.Event from an API error.
func (env apiEnvironment) ReportAPIError(err apiError) poll.Action {
	return func(poll.Enqueuer) error {
		return env.Callback(err.ToBeatEvent(), nil)
	}
}

func (env apiEnvironment) toBeatEvent(raw json.RawMessage, doc common.MapStr) beat.Event {
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
	}
	if env.Config.SetIDFromAuditRecord {
		if id, err := getString(doc, "Id"); err == nil && len(id) > 0 {
			b.SetID(id)
		}
	}
	if env.Config.PreserveOriginalEvent {
		b.PutValue("event.original", string(raw))
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
