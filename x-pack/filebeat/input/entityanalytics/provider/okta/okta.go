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
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
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
	lim    *rate.Limiter

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
	p.metrics = newMetrics(inputCtx.ID, nil)
	defer p.metrics.Close()

	lastSyncTime, _ := getLastSync(store)
	syncWaitTime := time.Until(lastSyncTime.Add(p.cfg.SyncInterval))
	lastUpdateTime, _ := getLastUpdate(store)
	updateWaitTime := time.Until(lastUpdateTime.Add(p.cfg.UpdateInterval))

	syncTimer := time.NewTimer(syncWaitTime)
	updateTimer := time.NewTimer(updateWaitTime)

	// Allow a single fetch operation to obtain limits from the API.
	p.lim = rate.NewLimiter(1, 1)

	var err error
	p.client, err = newClient(p.cfg, p.logger)
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

func newClient(cfg conf, log *logp.Logger) (*http.Client, error) {
	c, err := cfg.Request.Transport.Client(clientOptions(cfg.Request.KeepAlive.settings())...)
	if err != nil {
		return nil, err
	}

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

	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	p.logger.Debugf("Starting fetch...")
	_, err = p.doFetchUsers(ctx, state, true)
	if err != nil {
		return err
	}
	_, err = p.doFetchDevices(ctx, state, true)
	if err != nil {
		return err
	}

	wantUsers := p.cfg.wantUsers()
	wantDevices := p.cfg.wantDevices()
	if (len(state.users) != 0 && wantUsers) || (len(state.devices) != 0 && wantDevices) {
		tracker := kvstore.NewTxTracker(ctx)

		start := time.Now()
		p.publishMarker(start, start, inputCtx.ID, true, client, tracker)
		if wantUsers {
			for _, u := range state.users {
				p.publishUser(u, state, inputCtx.ID, client, tracker)
			}
		}
		if wantDevices {
			for _, d := range state.devices {
				p.publishDevice(d, state, inputCtx.ID, client, tracker)
			}
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
	updatedUsers, err := p.doFetchUsers(ctx, state, false)
	if err != nil {
		return err
	}
	updatedDevices, err := p.doFetchDevices(ctx, state, false)
	if err != nil {
		return err
	}

	var tracker *kvstore.TxTracker
	if len(updatedUsers) != 0 || len(updatedDevices) != 0 {
		tracker = kvstore.NewTxTracker(ctx)
		for _, u := range updatedUsers {
			p.publishUser(u, state, inputCtx.ID, client, tracker)
		}
		for _, d := range updatedDevices {
			p.publishDevice(d, state, inputCtx.ID, client, tracker)
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

// doFetchUsers handles fetching user identities from Okta. If fullSync is true, then
// any existing deltaLink will be ignored, forcing a full synchronization from Okta.
// Returns a set of modified users by ID.
func (p *oktaInput) doFetchUsers(ctx context.Context, state *stateStore, fullSync bool) ([]*User, error) {
	if !p.cfg.wantUsers() {
		p.logger.Debugf("Skipping user collection from API: dataset=%s", p.cfg.Dataset)
		return nil, nil
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

	const omit = okta.OmitCredentials | okta.OmitCredentialsLinks | okta.OmitTransitioningToStatus

	var (
		users       []*User
		lastUpdated time.Time
	)
	for {
		batch, h, err := okta.GetUserDetails(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, "", query, omit, p.lim, p.cfg.LimitWindow)
		if err != nil {
			p.logger.Debugf("received %d users from API", len(users))
			return nil, err
		}
		p.logger.Debugf("received batch of %d users from API", len(batch))

		if fullSync {
			for _, u := range batch {
				state.storeUser(u)
				if u.LastUpdated.After(lastUpdated) {
					lastUpdated = u.LastUpdated
				}
			}
		} else {
			users = grow(users, len(batch))
			for _, u := range batch {
				users = append(users, state.storeUser(u))
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
			p.logger.Debugf("received %d users from API", len(users))
			return users, err
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

	p.logger.Debugf("received %d users from API", len(users))
	return users, nil
}

// doFetchDevices handles fetching device and associated user identities from Okta.
// If fullSync is true, then any existing deltaLink will be ignored, forcing a full
// synchronization from Okta.
// Returns a set of modified devices by ID.
func (p *oktaInput) doFetchDevices(ctx context.Context, state *stateStore, fullSync bool) ([]*Device, error) {
	if !p.cfg.wantDevices() {
		p.logger.Debugf("Skipping device collection from API: dataset=%s", p.cfg.Dataset)
		return nil, nil
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
	// Start user queries from the same time point. This must not
	// be mutated since we may perform multiple batched gets over
	// multiple devices.
	userQueryInit = cloneURLValues(deviceQuery)

	var (
		devices     []*Device
		lastUpdated time.Time
	)
	for {
		batch, h, err := okta.GetDeviceDetails(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, "", deviceQuery, p.lim, p.cfg.LimitWindow)
		if err != nil {
			p.logger.Debugf("received %d devices from API", len(devices))
			return nil, err
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

				users, h, err := okta.GetDeviceUsers(ctx, p.client, p.cfg.OktaDomain, p.cfg.OktaToken, d.ID, userQuery, omit, p.lim, p.cfg.LimitWindow)
				if err != nil {
					p.logger.Debugf("received %d device users from API", len(users))
					return nil, err
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
					p.logger.Debugf("received %d devices from API", len(devices))
					return devices, err
				}
				userQuery = next
			}
		}

		if fullSync {
			for _, d := range batch {
				state.storeDevice(d)
				if d.LastUpdated.After(lastUpdated) {
					lastUpdated = d.LastUpdated
				}
			}
		} else {
			devices = grow(devices, len(batch))
			for _, d := range batch {
				devices = append(devices, state.storeDevice(d))
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
			p.logger.Debugf("received %d devices from API", len(devices))
			return devices, err
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

	p.logger.Debugf("received %d devices from API", len(devices))
	return devices, nil
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

func grow[T entity](e []T, n int) []T {
	if len(e)+n <= cap(e) {
		return e
	}
	new := append(e, make([]T, n)...)
	return new[:len(e)]
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
