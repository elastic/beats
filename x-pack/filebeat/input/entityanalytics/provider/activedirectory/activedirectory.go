// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package activedirectory provides a user identity asset provider for Microsoft
// Active Directory.
package activedirectory

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-ldap/ldap/v3"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/activedirectory/internal/activedirectory"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/go-concert/ctxtool"
)

func init() {
	err := provider.Register(Name, New)
	if err != nil {
		panic(err)
	}
}

// Name of this provider.
const Name = "activedirectory"

// FullName of this provider, including the input name. Prefer using this
// value for full context, especially if the input name isn't present in an
// adjacent log field.
const FullName = "entity-analytics-" + Name

// adInput implements the provider.Provider interface.
type adInput struct {
	*kvstore.Manager

	cfg       conf
	baseDN    *ldap.DN
	tlsConfig *tls.Config

	metrics *inputMetrics
	logger  *logp.Logger
}

// New creates a new instance of an Active Directory identity provider.
func New(logger *logp.Logger) (provider.Provider, error) {
	p := adInput{
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
func (p *adInput) configure(cfg *config.C) (kvstore.Input, error) {
	err := cfg.Unpack(&p.cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack %s input config: %w", Name, err)
	}
	p.baseDN, err = ldap.ParseDN(p.cfg.BaseDN)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(p.cfg.URL)
	if err != nil {
		return nil, err
	}
	if p.cfg.TLS.IsEnabled() && u.Scheme == "ldaps" {
		tlsConfig, err := tlscommon.LoadTLSConfig(p.cfg.TLS)
		if err != nil {
			return nil, err
		}
		host, _, err := net.SplitHostPort(u.Host)
		var addrErr *net.AddrError
		switch {
		case err == nil:
		case errors.As(err, &addrErr):
			if addrErr.Err != "missing port in address" {
				return nil, err
			}
			host = u.Host
		default:
			return nil, err
		}
		p.tlsConfig = tlsConfig.BuildModuleClientConfig(host)
	}
	return p, nil
}

// Name returns the name of this provider.
func (p *adInput) Name() string {
	return FullName
}

func (*adInput) Test(v2.TestContext) error { return nil }

// Run will start data collection on this provider.
func (p *adInput) Run(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger = inputCtx.Logger.With("provider", Name, "domain", p.cfg.URL)
	p.metrics = newMetrics(inputCtx.ID, nil)
	defer p.metrics.Close()

	lastSyncTime, _ := getLastSync(store)
	syncWaitTime := time.Until(lastSyncTime.Add(p.cfg.SyncInterval))
	lastUpdateTime, _ := getLastUpdate(store)
	updateWaitTime := time.Until(lastUpdateTime.Add(p.cfg.UpdateInterval))

	syncTimer := time.NewTimer(syncWaitTime)
	updateTimer := time.NewTimer(updateWaitTime)

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

// clientOption returns constructed client configuration options, including
// setting up http+unix and http+npipe transports if requested.
func clientOptions(keepalive httpcommon.WithKeepaliveSettings) []httpcommon.TransportOption {
	return []httpcommon.TransportOption{
		httpcommon.WithAPMHTTPInstrumentation(),
		keepalive,
	}
}

// runFullSync performs a full synchronization. It will fetch user and group
// identities from Azure Active Directory, enrich users with group memberships,
// and publishes all known users (regardless if they have been modified) to the
// given beat.Client.
func (p *adInput) runFullSync(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
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

	if len(state.users) != 0 {
		tracker := kvstore.NewTxTracker(ctx)

		start := time.Now()
		p.publishMarker(start, start, inputCtx.ID, true, client, tracker)
		for _, u := range state.users {
			p.publishUser(u, state, inputCtx.ID, client, tracker)
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
func (p *adInput) runIncrementalUpdate(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
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

	var tracker *kvstore.TxTracker
	if len(updatedUsers) != 0 || state.len() != 0 {
		// Active Directory does not have a notion of deleted users
		// beyond absence from the directory, so compare found users
		// with users already known by the state store and if any
		// are in the store but not returned in the previous fetch,
		// mark them as deleted and publish the deletion. We do not
		// have the time of the deletion, so use now.
		if state.len() != 0 {
			found := make(map[string]bool)
			for _, u := range updatedUsers {
				found[u.ID] = true
			}
			deleted := make(map[string]*User)
			now := time.Now()
			state.forEach(func(u *User) {
				if u.State == Deleted || found[u.ID] {
					return
				}
				// This modifies the state store's copy since u
				// is a pointer held by the state store map.
				u.State = Deleted
				u.WhenChanged = now
				deleted[u.ID] = u
			})
			for _, u := range deleted {
				updatedUsers = append(updatedUsers, u)
			}
		}
		if len(updatedUsers) != 0 {
			tracker = kvstore.NewTxTracker(ctx)
			for _, u := range updatedUsers {
				p.publishUser(u, state, inputCtx.ID, client, tracker)
			}
			tracker.Wait()
		}
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

// doFetchUsers handles fetching user identities from Active Directory. If
// fullSync is true, then any existing whenChanged will be ignored, forcing a
// full synchronization from Active Directory.
// Returns a set of modified users by ID.
func (p *adInput) doFetchUsers(ctx context.Context, state *stateStore, fullSync bool) ([]*User, error) {
	var since time.Time
	if !fullSync {
		since = state.whenChanged
	}

	entries, err := activedirectory.GetDetails(p.cfg.URL, p.cfg.User, p.cfg.Password, p.baseDN, since, p.cfg.PagingSize, nil, p.tlsConfig)
	p.logger.Debugf("received %d users from API", len(entries))
	if err != nil {
		return nil, err
	}

	var (
		users       []*User
		whenChanged time.Time
	)
	if fullSync {
		for _, u := range entries {
			state.storeUser(u)
			if u.WhenChanged.After(whenChanged) {
				whenChanged = u.WhenChanged
			}
		}
	} else {
		users = make([]*User, 0, len(entries))
		for _, u := range entries {
			users = append(users, state.storeUser(u))
			if u.WhenChanged.After(whenChanged) {
				whenChanged = u.WhenChanged
			}
		}
		p.logger.Debugf("processed %d users from API", len(users))
	}
	if whenChanged.After(state.whenChanged) {
		state.whenChanged = whenChanged
	}

	return users, nil
}

// publishMarker will publish a write marker document using the given beat.Client.
// If start is true, then it will be a start marker, otherwise an end marker.
func (p *adInput) publishMarker(ts, eventTime time.Time, inputID string, start bool, client beat.Client, tracker *kvstore.TxTracker) {
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
func (p *adInput) publishUser(u *User, state *stateStore, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	userDoc := mapstr.M{}

	_, _ = userDoc.Put("activedirectory", u.User)
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
