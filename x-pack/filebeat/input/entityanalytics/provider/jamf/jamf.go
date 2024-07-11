// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package jamf provides a computer asset provider for Jamf.
package jamf

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/jamf/internal/jamf"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/go-concert/ctxtool"
)

func init() {
	err := provider.Register(Name, New)
	if err != nil {
		panic(err)
	}
}

// Name of this provider.
const Name = "jamf"

// FullName of this provider, including the input name. Prefer using this
// value for full context, especially if the input name isn't present in an
// adjacent log field.
const FullName = "entity-analytics-" + Name

// jamfInput implements the provider.Provider interface.
type jamfInput struct {
	*kvstore.Manager

	cfg conf

	client *http.Client
	token  jamf.Token

	metrics *inputMetrics
	logger  *logp.Logger
}

// New creates a new instance of an Jamf entity provider.
func New(logger *logp.Logger) (provider.Provider, error) {
	p := jamfInput{
		cfg: defaultConfig(),
	}
	p.Manager = &kvstore.Manager{
		Logger:    logger,
		Type:      FullName,
		Configure: p.configure,
	}

	return &p, nil
}

// configure configures this provider using the given configuration.
func (p *jamfInput) configure(cfg *config.C) (kvstore.Input, error) {
	err := cfg.Unpack(&p.cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack %s input config: %w", Name, err)
	}
	return p, nil
}

// Name returns the name of this provider.
func (p *jamfInput) Name() string {
	return FullName
}

func (*jamfInput) Test(v2.TestContext) error { return nil }

// Run will start data collection on this provider.
func (p *jamfInput) Run(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger = inputCtx.Logger.With("provider", Name, "tenant", p.cfg.JamfTenant)
	p.metrics = newMetrics(inputCtx.ID, nil)
	defer p.metrics.Close()

	lastSyncTime, _ := getLastSync(store)
	syncWaitTime := time.Until(lastSyncTime.Add(p.cfg.SyncInterval))
	lastUpdateTime, _ := getLastUpdate(store)
	updateWaitTime := time.Until(lastUpdateTime.Add(p.cfg.UpdateInterval))

	syncTimer := time.NewTimer(syncWaitTime)
	updateTimer := time.NewTimer(updateWaitTime)

	if p.cfg.Tracer != nil {
		id := sanitizeFileName(inputCtx.ID)
		p.cfg.Tracer.Filename = strings.ReplaceAll(p.cfg.Tracer.Filename, "*", id)
	}

	var err error
	p.client, err = newClient(ctxtool.FromCanceller(inputCtx.Cancelation), p.cfg, p.logger)
	if err != nil {
		return err
	}

	for {
		select {
		case <-inputCtx.Cancelation.Done():
			if !errors.Is(inputCtx.Cancelation.Err(), context.Canceled) {
				return inputCtx.Cancelation.Err()
			}
			return nil
		case <-syncTimer.C:
			start := time.Now()
			if err := p.runFullSync(inputCtx, store, client); err != nil {
				p.logger.Errorw("Error running full sync", "error", err)
				p.metrics.syncError.Inc()
			}
			p.metrics.syncTotal.Inc()
			p.metrics.syncProcessingTime.Update(time.Since(start).Nanoseconds())

			syncTimer.Reset(p.cfg.SyncInterval)
			p.logger.Debugf("Next sync expected at: %v", time.Now().Add(p.cfg.SyncInterval))

			// Reset the update timer and wait the configured interval. If the
			// update timer has already fired, then drain the timer's channel
			// before resetting.
			if !updateTimer.Stop() {
				<-updateTimer.C
			}
			updateTimer.Reset(p.cfg.UpdateInterval)
			p.logger.Debugf("Next update expected at: %v", time.Now().Add(p.cfg.UpdateInterval))
		case <-updateTimer.C:
			start := time.Now()
			if err := p.runIncrementalUpdate(inputCtx, store, client); err != nil {
				p.logger.Errorw("Error running incremental update", "error", err)
				p.metrics.updateError.Inc()
			}
			p.metrics.updateTotal.Inc()
			p.metrics.updateProcessingTime.Update(time.Since(start).Nanoseconds())
			updateTimer.Reset(p.cfg.UpdateInterval)
			p.logger.Debugf("Next update expected at: %v", time.Now().Add(p.cfg.UpdateInterval))
		}
	}
}

func newClient(ctx context.Context, cfg conf, log *logp.Logger) (*http.Client, error) {
	c, err := cfg.Request.Transport.Client(clientOptions(cfg.Request.KeepAlive.settings())...)
	if err != nil {
		return nil, err
	}

	c = requestTrace(ctx, c, cfg, log)

	c.CheckRedirect = checkRedirect(cfg.Request, log)

	client := &retryablehttp.Client{
		HTTPClient:   c,
		Logger:       newRetryLog(log),
		RetryWaitMin: cfg.Request.Retry.getWaitMin(),
		RetryWaitMax: cfg.Request.Retry.getWaitMax(),
		RetryMax:     cfg.Request.Retry.getMaxAttempts(),
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}
	return client.StandardClient(), nil
}

// lumberjackTimestamp is a glob expression matching the time format string used
// by lumberjack when rolling over logs, "2006-01-02T15-04-05.000".
// https://github.com/natefinch/lumberjack/blob/4cb27fcfbb0f35cb48c542c5ea80b7c1d18933d0/lumberjack.go#L39
const lumberjackTimestamp = "[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]-[0-9][0-9]-[0-9][0-9].[0-9][0-9][0-9]"

// requestTrace decorates cli with an httplog.LoggingRoundTripper if cfg.Tracer
// is non-nil.
func requestTrace(ctx context.Context, cli *http.Client, cfg conf, log *logp.Logger) *http.Client {
	if cfg.Tracer == nil {
		return cli
	}
	if !cfg.Tracer.enabled() {
		// We have a trace log name, but we are not enabled,
		// so remove all trace logs we own.
		err := os.Remove(cfg.Tracer.Filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Errorw("failed to remove request trace log", "path", cfg.Tracer.Filename, "error", err)
		}
		ext := filepath.Ext(cfg.Tracer.Filename)
		base := strings.TrimSuffix(cfg.Tracer.Filename, ext)
		paths, err := filepath.Glob(base + "-" + lumberjackTimestamp + ext)
		if err != nil {
			log.Errorw("failed to collect request trace log path names", "error", err)
		}
		for _, p := range paths {
			err = os.Remove(p)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Errorw("failed to remove request trace log", "path", p, "error", err)
			}
		}
		return cli
	}

	w := zapcore.AddSync(cfg.Tracer)
	go func() {
		// Close the logger when we are done.
		<-ctx.Done()
		cfg.Tracer.Close()
	}()
	core := ecszap.NewCore(
		ecszap.NewDefaultEncoderConfig(),
		w,
		zap.DebugLevel,
	)
	traceLogger := zap.New(core)

	const margin = 10e3 // 1OkB ought to be enough room for all the remainder of the trace details.
	maxSize := cfg.Tracer.MaxSize * 1e6
	cli.Transport = httplog.NewLoggingRoundTripper(cli.Transport, traceLogger, max(0, maxSize-margin), log)
	return cli
}

// sanitizeFileName returns name with ":" and "/" replaced with "_", removing
// repeated instances. The request.tracer.filename may have ":" when an input
// has cursor config and the macOS Finder will treat this as path-separator and
// causes to show up strange filepaths.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, ":", string(filepath.Separator))
	name = filepath.Clean(name)
	return strings.ReplaceAll(name, string(filepath.Separator), "_")
}

// clientOption returns constructed client configuration options, including
// setting up http+unix and http+npipe transports if requested.
func clientOptions(keepalive httpcommon.WithKeepaliveSettings) []httpcommon.TransportOption {
	return []httpcommon.TransportOption{
		httpcommon.WithAPMHTTPInstrumentation(),
		keepalive,
	}
}

func checkRedirect(cfg *requestConfig, log *logp.Logger) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		log.Debug("http client: checking redirect")
		if len(via) >= cfg.RedirectMaxRedirects {
			log.Debug("http client: max redirects exceeded")
			return fmt.Errorf("stopped after %d redirects", cfg.RedirectMaxRedirects)
		}

		if !cfg.RedirectForwardHeaders || len(via) == 0 {
			log.Debugf("http client: nothing to do while checking redirects - forward_headers: %v, via: %#v", cfg.RedirectForwardHeaders, via)
			return nil
		}

		prev := via[len(via)-1] // previous request to get headers from

		log.Debugf("http client: forwarding headers from previous request: %#v", prev.Header)
		req.Header = prev.Header.Clone()

		for _, k := range cfg.RedirectHeadersBanList {
			log.Debugf("http client: ban header %v", k)
			req.Header.Del(k)
		}

		return nil
	}
}

// retryLog is a shim for the retryablehttp.Client.Logger.
type retryLog struct{ log *logp.Logger }

func newRetryLog(log *logp.Logger) *retryLog {
	return &retryLog{log: log.Named("retryablehttp").WithOptions(zap.AddCallerSkip(1))}
}

func (l *retryLog) Error(msg string, kv ...interface{}) { l.log.Errorw(msg, kv...) }
func (l *retryLog) Info(msg string, kv ...interface{})  { l.log.Infow(msg, kv...) }
func (l *retryLog) Debug(msg string, kv ...interface{}) { l.log.Debugw(msg, kv...) }
func (l *retryLog) Warn(msg string, kv ...interface{})  { l.log.Warnw(msg, kv...) }

// runFullSync performs a full synchronization. It will fetch user and group
// identities from Azure Active Directory, enrich users with group memberships,
// and publishes all known users (regardless if they have been modified) to the
// given beat.Client.
func (p *jamfInput) runFullSync(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger.Debugf("Running full sync...")

	p.logger.Debugf("Opening new transaction...")
	state, err := newStateStore(store)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	p.logger.Debugf("Transaction opened")
	defer func() { // If commit is successful, call to this close will be no-op.
		closeErr := state.close(false)
		if closeErr != nil {
			p.logger.Errorw("Error rolling back full sync transaction", "error", closeErr)
		}
	}()

	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	p.logger.Debugf("Starting fetch...")
	_, err = p.doFetchComputers(ctx, state, true)
	if err != nil {
		return err
	}

	if len(state.computers) != 0 {
		tracker := kvstore.NewTxTracker(ctx)

		start := time.Now()
		p.publishMarker(start, start, inputCtx.ID, true, client, tracker)
		for _, c := range state.computers {
			p.publishComputer(c, inputCtx.ID, client, tracker)
		}

		end := time.Now()
		p.publishMarker(end, end, inputCtx.ID, false, client, tracker)

		tracker.Wait()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	state.lastSync = time.Now()
	err = state.close(true)
	if err != nil {
		return fmt.Errorf("unable to commit state: %w", err)
	}

	return nil
}

// runIncrementalUpdate will run an incremental update. The process is similar
// to full synchronization, except only users which have changed (newly
// discovered, modified, or deleted) will be published.
func (p *jamfInput) runIncrementalUpdate(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger.Debugf("Running incremental update...")

	state, err := newStateStore(store)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer func() { // If commit is successful, call to this close will be no-op.
		closeErr := state.close(false)
		if closeErr != nil {
			p.logger.Errorw("Error rolling back incremental update transaction", "error", closeErr)
		}
	}()

	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	updatedDevices, err := p.doFetchComputers(ctx, state, false)
	if err != nil {
		return err
	}

	var tracker *kvstore.TxTracker
	if len(updatedDevices) != 0 {
		tracker = kvstore.NewTxTracker(ctx)
		for _, d := range updatedDevices {
			p.publishComputer(d, inputCtx.ID, client, tracker)
		}
		tracker.Wait()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	state.lastUpdate = time.Now()
	if err = state.close(true); err != nil {
		return fmt.Errorf("unable to commit state: %w", err)
	}

	return nil
}

// doFetchComputers handles fetching computer and associated user identities from Jamf.
// If a full synchronization from Jamf is performed.
// Returns a set of modified devices by ID.
func (p *jamfInput) doFetchComputers(ctx context.Context, state *stateStore, fullSync bool) ([]*Computer, error) {
	var (
		computers []*Computer
		query     url.Values
		err       error
	)
	if p.cfg.PageSize > 0 {
		query = make(url.Values)
		query.Set("page-size", strconv.Itoa(p.cfg.PageSize))
	}
	for page, n := 0, 0; ; page++ {
		if !p.token.IsValidFor(p.cfg.TokenGrace) {
			p.token, err = jamf.GetToken(ctx, p.client, p.cfg.JamfTenant, p.cfg.JamfUsername, p.cfg.JamfPassword)
			if err != nil {
				return nil, fmt.Errorf("failed to get auth token: %w", err)
			}
		}

		if query != nil {
			query.Set("page", strconv.Itoa(page))
		}
		resp, err := jamf.GetComputers(ctx, p.client, p.cfg.JamfTenant, p.token, query)
		if err != nil {
			p.logger.Debugf("received %d computers from API", len(computers))
			return nil, err
		}
		if len(resp.Results) == 0 {
			break
		}
		p.logger.Debugf("received batch of %d computers from API", len(resp.Results))

		if fullSync {
			for _, c := range resp.Results {
				state.storeComputer(c)
			}
		} else {
			for _, c := range resp.Results {
				stored, changed := state.storeComputer(c)
				if stored == nil {
					continue
				}
				if changed {
					computers = append(computers, stored)
				}
			}
		}

		n += len(resp.Results)
		if n >= resp.TotalCount {
			break
		}
	}

	p.logger.Debugf("received %d modified computer records from API", len(computers))
	return computers, nil
}

// publishMarker will publish a write marker document using the given beat.Client.
// If start is true, then it will be a start marker, otherwise an end marker.
func (p *jamfInput) publishMarker(ts, eventTime time.Time, inputID string, start bool, client beat.Client, tracker *kvstore.TxTracker) {
	fields := mapstr.M{}
	_, _ = fields.Put("labels.identity_source", inputID)

	if start {
		_, _ = fields.Put("event.action", "started")
		_, _ = fields.Put("event.start", eventTime)
	} else {
		_, _ = fields.Put("event.action", "completed")
		_, _ = fields.Put("event.end", eventTime)
	}

	event := beat.Event{
		Timestamp: ts,
		Fields:    fields,
		Private:   tracker,
	}
	tracker.Add()
	if start {
		p.logger.Debug("Publishing start write marker")
	} else {
		p.logger.Debug("Publishing end write marker")
	}

	client.Publish(event)
}

// publishComputer will publish a computer document using the given beat.Client.
func (p *jamfInput) publishComputer(c *Computer, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	devDoc := mapstr.M{}

	id := "unknown"
	if c.Udid != nil {
		id = *c.Udid
	}
	_, _ = devDoc.Put("jamf", c.Computer)
	_, _ = devDoc.Put("labels.identity_source", inputID)
	_, _ = devDoc.Put("device.id", id)

	switch c.State {
	case Deleted:
		_, _ = devDoc.Put("event.action", "device-deleted")
	case Discovered:
		_, _ = devDoc.Put("event.action", "device-discovered")
	case Modified:
		_, _ = devDoc.Put("event.action", "device-modified")
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    devDoc,
		Private:   tracker,
	}
	tracker.Add()

	p.logger.Debugf("Publishing computer %q", id)

	client.Publish(event)
}
