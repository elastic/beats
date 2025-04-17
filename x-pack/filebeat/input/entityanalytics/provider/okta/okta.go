// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package okta provides a user identity asset provider for Okta.
package okta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
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
const Name = "okta"

// FullName of this provider, including the input name. Prefer using this
// value for full context, especially if the input name isn't present in an
// adjacent log field.
const FullName = "entity-analytics-" + Name

// oktaInput implements the provider.Provider interface.
type oktaInput struct {
	*kvstore.Manager

	cfg conf

	client *http.Client
	lim    *okta.RateLimiter

	metrics *inputMetrics
	logger  *logp.Logger
}

// New creates a new instance of an Okta identity provider.
func New(logger *logp.Logger) (provider.Provider, error) {
	p := oktaInput{
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
func (p *oktaInput) configure(cfg *config.C) (kvstore.Input, error) {
	err := cfg.Unpack(&p.cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack %s input config: %w", Name, err)
	}
	return p, nil
}

// Name returns the name of this provider.
func (p *oktaInput) Name() string {
	return FullName
}

func (*oktaInput) Test(v2.TestContext) error { return nil }

// Run will start data collection on this provider.
func (p *oktaInput) Run(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger = inputCtx.Logger.With("provider", Name, "domain", p.cfg.OktaDomain)
	p.metrics = newMetrics(inputCtx)

	lastSyncTime, _ := getLastSync(store)
	syncWaitTime := time.Until(lastSyncTime.Add(p.cfg.SyncInterval))
	lastUpdateTime, _ := getLastUpdate(store)
	updateWaitTime := time.Until(lastUpdateTime.Add(p.cfg.UpdateInterval))

	syncTimer := time.NewTimer(syncWaitTime)
	updateTimer := time.NewTimer(updateWaitTime)

	// Allow a single fetch operation to obtain limits from the API.
	p.lim = okta.NewRateLimiter(p.cfg.LimitWindow, p.cfg.LimitFixed)

	if p.cfg.Tracer != nil {
		id := sanitizeFileName(inputCtx.IDWithoutName)
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

	maxBodyLen := cfg.Tracer.MaxSize * 1e6 / 10 // 10% of file max
	cli.Transport = httplog.NewLoggingRoundTripper(cli.Transport, traceLogger, maxBodyLen, log)
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
func (p *oktaInput) runFullSync(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
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

	wantUsers := p.cfg.wantUsers()
	wantDevices := p.cfg.wantDevices()
	if wantUsers || wantDevices {
		ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
		p.logger.Debugf("Starting fetch...")

		tracker := kvstore.NewTxTracker(ctx)

		start := time.Now()
		p.publishMarker(start, start, inputCtx.ID, true, client, tracker)

		if wantUsers {
			err = p.doFetchUsers(ctx, state, true, func(u *User) {
				p.publishUser(u, state, inputCtx.ID, client, tracker)
			})
			if err != nil {
				return err
			}
		}
		if wantDevices {
			err = p.doFetchDevices(ctx, state, true, func(d *Device) {
				p.publishDevice(d, state, inputCtx.ID, client, tracker)
			})
			if err != nil {
				return err
			}
		}

		end := time.Now()
		p.publishMarker(end, end, inputCtx.ID, false, client, tracker)

		tracker.Wait()

		if ctx.Err() != nil {
			return ctx.Err()
		}
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
func (p *oktaInput) runIncrementalUpdate(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
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
	tracker := kvstore.NewTxTracker(ctx)

	if p.cfg.wantUsers() {
		p.logger.Debugf("Fetching changed users...")
		err = p.doFetchUsers(ctx, state, false, func(u *User) {
			p.publishUser(u, state, inputCtx.ID, client, tracker)
		})
		if err != nil {
			return err
		}
	}
	if p.cfg.wantDevices() {
		p.logger.Debugf("Fetching changed devices...")
		err = p.doFetchDevices(ctx, state, false, func(d *Device) {
			p.publishDevice(d, state, inputCtx.ID, client, tracker)
		})
		if err != nil {
			return err
		}
	}

	tracker.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}

	state.lastUpdate = time.Now()
	if err = state.close(true); err != nil {
		return fmt.Errorf("unable to commit state: %w", err)
	}

	return nil
}

// doFetchUsers handles fetching user identities from Okta. If fullSync is true, then
// any existing deltaLink will be ignored, forcing a full synchronization from Okta.
// Returns a set of modified users by ID.
func (p *oktaInput) doFetchUsers(ctx context.Context, state *stateStore, fullSync bool, publish func(u *User)) error {
	if !p.cfg.wantUsers() {
		p.logger.Debugf("Skipping user collection from API: dataset=%s", p.cfg.Dataset)
		return nil
	}

	var (
		query url.Values
		err   error
	)

	// Get user changes.
	if !fullSync && state.nextUsers != "" {
		query, err = url.ParseQuery(state.nextUsers)
		if err != nil {
			p.logger.Warnf("failed to parse next query: %v", err)
		}
	}
	if query == nil {
		// Use "search" because of recommendation on Okta dev documentation:
		// https://developer.okta.com/docs/reference/user-query/.
		// Search term of "status pr" is required so that we get DEPROVISIONED
		// users; a nil query is more efficient, but excludes these users.
		query = url.Values{"search": []string{"status pr"}}
	}
	if p.cfg.BatchSize > 0 {
		// If limit is not specified, the API default is used in the case
		// that we are using, this is 200.
		//
		// See:
		//  https://developer.okta.com/docs/api/openapi/okta-management/management/tag/User/#tag/User/operation/listUsers!in=query&path=limit&t=request
		query.Set("limit", strconv.Itoa(p.cfg.BatchSize))
	}

	const omit = okta.OmitCredentials | okta.OmitCredentialsLinks | okta.OmitTransitioningToStatus

	var (
		n           int
		lastUpdated time.Time
	)
	for {
		batch, h, err := okta.GetUserDetails(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, "", query, omit, p.lim, p.logger)
		if err != nil {
			p.logger.Debugf("received %d users from API", n)
			return err
		}
		p.logger.Debugf("received batch of %d users from API", len(batch))

		if fullSync {
			for _, u := range batch {
				publish(p.addUserMetadata(ctx, u, state))
				if u.LastUpdated.After(lastUpdated) {
					lastUpdated = u.LastUpdated
				}
			}
		} else {
			for _, u := range batch {
				su := p.addUserMetadata(ctx, u, state)
				publish(su)
				n++
				if u.LastUpdated.After(lastUpdated) {
					lastUpdated = u.LastUpdated
				}
			}
		}

		next, err := okta.Next(h)
		if err != nil {
			if err == io.EOF {
				break
			}
			p.logger.Debugf("received %d users from API", n)
			return err
		}
		query = next
	}

	// Prepare query for next update. This is any record that was updated
	// at or after the last updated record we saw this round. Use this rather
	// than time.Now() since we may have received stale records. Use ge
	// rather than gt since timestamps are second resolution, so we may not
	// have a complete set from that timestamp.
	query = url.Values{}
	query.Add("search", fmt.Sprintf(`lastUpdated ge "%s" and status pr`, lastUpdated.Format(okta.ISO8601)))
	state.nextUsers = query.Encode()

	p.logger.Debugf("received %d users from API", n)
	return nil
}

func (p *oktaInput) addUserMetadata(ctx context.Context, u okta.User, state *stateStore) *User {
	su := state.storeUser(u)
	switch len(p.cfg.EnrichWith) {
	case 1:
		if p.cfg.EnrichWith[0] != "none" {
			break
		}
		fallthrough
	case 0:
		return su
	}
	if slices.Contains(p.cfg.EnrichWith, "groups") {
		groups, _, err := okta.GetUserGroupDetails(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, u.ID, p.lim, p.logger)
		if err != nil {
			p.logger.Warnf("failed to get user group membership for %s: %v", u.ID, err)
		} else {
			su.Groups = groups
		}
	}
	if slices.Contains(p.cfg.EnrichWith, "factors") {
		factors, _, err := okta.GetUserFactors(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, u.ID, p.lim, p.logger)
		if err != nil {
			p.logger.Warnf("failed to get user factors for %s: %v", u.ID, err)
		} else {
			su.Factors = factors
		}
	}
	if slices.Contains(p.cfg.EnrichWith, "roles") {
		roles, _, err := okta.GetUserRoles(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, u.ID, p.lim, p.logger)
		if err != nil {
			p.logger.Warnf("failed to get user roles for %s: %v", u.ID, err)
		} else {
			su.Roles = roles
		}
	}
	return su
}

// doFetchDevices handles fetching device and associated user identities from Okta.
// If fullSync is true, then any existing deltaLink will be ignored, forcing a full
// synchronization from Okta.
// Returns a set of modified devices by ID.
func (p *oktaInput) doFetchDevices(ctx context.Context, state *stateStore, fullSync bool, publish func(d *Device)) error {
	if !p.cfg.wantDevices() {
		p.logger.Debugf("Skipping device collection from API: dataset=%s", p.cfg.Dataset)
		return nil
	}

	var (
		deviceQuery   url.Values
		userQueryInit url.Values
		err           error
	)

	// Get user changes.
	if !fullSync && state.nextDevices != "" {
		deviceQuery, err = url.ParseQuery(state.nextDevices)
		if err != nil {
			p.logger.Warnf("failed to parse next query: %v", err)
		}
	}
	if deviceQuery == nil {
		// Use "search" because of recommendation on Okta dev documentation:
		// https://developer.okta.com/docs/reference/user-query/.
		// Search term of "status pr" is required so that we get DEPROVISIONED
		// users; a nil query is more efficient, but excludes these users.
		// There is no equivalent documentation for devices, so we assume the
		// behaviour is the same.
		deviceQuery = url.Values{"search": []string{"status pr"}}
	}
	if p.cfg.BatchSize > 0 {
		// If limit is not specified, the API default is used in the case
		// that we are using, this is 200.
		//
		// See:
		//  https://developer.okta.com/docs/api/openapi/okta-management/management/tag/User/#tag/User/operation/listUsers!in=query&path=limit&t=request
		deviceQuery.Set("limit", strconv.Itoa(p.cfg.BatchSize))
	}
	// Start user queries from the same time point. This must not
	// be mutated since we may perform multiple batched gets over
	// multiple devices.
	userQueryInit = cloneURLValues(deviceQuery)

	var (
		n           int
		lastUpdated time.Time
	)
	for {
		batch, h, err := okta.GetDeviceDetails(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, "", deviceQuery, p.lim, p.logger)
		if err != nil {
			p.logger.Debugf("received %d devices from API", n)
			return err
		}
		p.logger.Debugf("received batch of %d devices from API", len(batch))

		for i, d := range batch {
			userQuery := cloneURLValues(userQueryInit)
			for {
				// TODO: Consider softening the response to errors here. If we fail to get users
				// from a device, do we want to fail completely? There are arguments in both
				// directions. We _could_ keep a multierror and return that in the end, which
				// would guarantee progression, but may result in holes in the data. What we are
				// doing at the moment (both here and in doFetchUsers) guarantees no holes, but
				// at the cost of potentially not making progress.

				const omit = okta.OmitCredentials | okta.OmitCredentialsLinks | okta.OmitTransitioningToStatus

				users, h, err := okta.GetDeviceUsers(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, d.ID, userQuery, omit, p.lim, p.logger)
				if err != nil {
					p.logger.Debugf("received %d device users from API", len(users))
					return err
				}
				p.logger.Debugf("received batch of %d device users from API", len(users))

				// Users are not stored in the state as they are in doFetchUsers. We expect
				// them to already have been discovered/stored from that call and are stored
				// associated with the device undecorated with discovery state. Or, if the
				// the dataset is set to "devices", then we have been asked not to care about
				// this detail.
				batch[i].Users = append(batch[i].Users, users...)

				next, err := okta.Next(h)
				if err != nil {
					if err == io.EOF {
						break
					}
					p.logger.Debugf("received %d devices from API", n)
					return err
				}
				userQuery = next
			}
		}

		if fullSync {
			for _, d := range batch {
				publish(state.storeDevice(d))
				if d.LastUpdated.After(lastUpdated) {
					lastUpdated = d.LastUpdated
				}
			}
		} else {
			for _, d := range batch {
				sd := state.storeDevice(d)
				publish(sd)
				n++
				if d.LastUpdated.After(lastUpdated) {
					lastUpdated = d.LastUpdated
				}
			}
		}

		next, err := okta.Next(h)
		if err != nil {
			if err == io.EOF {
				break
			}
			p.logger.Debugf("received %d devices from API", n)
			return err
		}
		deviceQuery = next
	}

	// Prepare query for next update. This is any record that was updated
	// at or after the last updated record we saw this round. Use this rather
	// than time.Now() since we may have received stale records. Use ge
	// rather than gt since timestamps are second resolution, so we may not
	// have a complete set from that timestamp.
	deviceQuery = url.Values{}
	deviceQuery.Add("search", fmt.Sprintf(`lastUpdated ge "%s" and status pr`, lastUpdated.Format(okta.ISO8601)))
	state.nextDevices = deviceQuery.Encode()

	p.logger.Debugf("received %d devices from API", n)
	return nil
}

func cloneURLValues(a url.Values) url.Values {
	b := make(url.Values, len(a))
	for k, v := range a {
		b[k] = append(v[:0:0], v...)
	}
	return b
}

type entity interface {
	*User | *Device | okta.User
}

// publishMarker will publish a write marker document using the given beat.Client.
// If start is true, then it will be a start marker, otherwise an end marker.
func (p *oktaInput) publishMarker(ts, eventTime time.Time, inputID string, start bool, client beat.Client, tracker *kvstore.TxTracker) {
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

// publishUser will publish a user document using the given beat.Client.
func (p *oktaInput) publishUser(u *User, state *stateStore, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	userDoc := mapstr.M{}

	_, _ = userDoc.Put("okta", u.User)
	_, _ = userDoc.Put("labels.identity_source", inputID)
	_, _ = userDoc.Put("user.id", u.ID)
	_, _ = userDoc.Put("groups", u.Groups)

	switch u.State {
	case Deleted:
		_, _ = userDoc.Put("event.action", "user-deleted")
	case Discovered:
		_, _ = userDoc.Put("event.action", "user-discovered")
	case Modified:
		_, _ = userDoc.Put("event.action", "user-modified")
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    userDoc,
		Private:   tracker,
	}
	tracker.Add()

	p.logger.Debugf("Publishing user %q", u.ID)

	client.Publish(event)
}

// publishDevice will publish a device document using the given beat.Client.
func (p *oktaInput) publishDevice(d *Device, state *stateStore, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	devDoc := mapstr.M{}

	_, _ = devDoc.Put("okta", d.Device)
	_, _ = devDoc.Put("labels.identity_source", inputID)
	_, _ = devDoc.Put("device.id", d.ID)

	switch d.State {
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

	p.logger.Debugf("Publishing device %q", d.ID)

	client.Publish(event)
}
