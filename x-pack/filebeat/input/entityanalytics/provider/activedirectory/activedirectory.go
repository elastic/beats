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

	p.cfg.UserAttrs = withMandatory(p.cfg.UserAttrs, "distinguishedName", "whenChanged")
	p.cfg.GrpAttrs = withMandatory(p.cfg.GrpAttrs, "distinguishedName", "whenChanged")

	var (
		last time.Time
		err  error
	)
	for {
		select {
		case <-inputCtx.Cancelation.Done():
			if !errors.Is(inputCtx.Cancelation.Err(), context.Canceled) {
				return inputCtx.Cancelation.Err()
			}
			return nil
		case start := <-syncTimer.C:
			last, err = p.runFullSync(inputCtx, store, client)
			if err != nil {
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
		case start := <-updateTimer.C:
			last, err = p.runIncrementalUpdate(inputCtx, store, last, client)
			if err != nil {
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

// withMandatory adds the required attribute names to attr unless attr is empty.
func withMandatory(attr []string, include ...string) []string {
	if len(attr) == 0 {
		return nil
	}
outer:
	for _, m := range include {
		for _, a := range attr {
			if m == a {
				continue outer
			}
		}
		attr = append(attr, m)
	}
	return attr
}

// runFullSync performs a full synchronization. It will fetch user and group
// identities from Azure Active Directory, enrich users with group memberships,
// and publishes all known users (regardless if they have been modified) to the
// given beat.Client.
func (p *adInput) runFullSync(inputCtx v2.Context, store *kvstore.Store, client beat.Client) (time.Time, error) {
	p.logger.Debugf("Running full sync...")

	p.logger.Debugf("Opening new transaction...")
	state, err := newStateStore(store)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to begin transaction: %w", err)
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
		var users, devices []*User
		ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
		p.logger.Debugf("Starting fetch...")
		if wantUsers {
			users, err = p.doFetchUsers(ctx, state, true)
			if err != nil {
				return time.Time{}, err
			}
		}
		if wantDevices {
			devices, err = p.doFetchDevices(ctx, state, true)
			if err != nil {
				return time.Time{}, err
			}
		}

		tracker := kvstore.NewTxTracker(ctx)
		start := time.Now()
		p.publishMarker(start, start, inputCtx.ID, true, client, tracker)

		for _, u := range p.unifyState(ctx, state.users, users) {
			p.publishUser(u, state, inputCtx.ID, client, tracker)
		}
		for _, d := range p.unifyState(ctx, state.devices, devices) {
			p.publishDevice(d, state, inputCtx.ID, client, tracker)
		}

		end := time.Now()
		p.publishMarker(end, end, inputCtx.ID, false, client, tracker)
		tracker.Wait()

		if ctx.Err() != nil {
			return time.Time{}, ctx.Err()
		}
	}

	// state.whenChanged is modified by the call to doFetchUsers to be
	// the latest modification time for all of the users that have been
	// collected in that call. This will not include any of the deleted
	// users since they were not collected.
	latest := state.whenChanged
	state.lastSync = latest
	err = state.close(true)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to commit state: %w", err)
	}

	return latest, nil
}

// unifyState merges the state and entries, updating User values that have
// are in state, but not in entries to mark them as deleted.
func (p *adInput) unifyState(ctx context.Context, state map[string]*User, entries []*User) []*User {
	if len(entries) == 0 && len(state) == 0 {
		return nil
	}

	// Active Directory does not have a notion of deleted users
	// beyond absence from the directory, so compare found users
	// with users already known by the state store and if any
	// are in the store but not returned in the previous fetch,
	// mark them as deleted and publish the deletion. We do not
	// have the time of the deletion, so use now.
	if len(state) != 0 {
		found := make(map[string]bool)
		for _, u := range entries {
			found[u.ID] = true
		}
		deleted := make(map[string]*User)
		now := time.Now()
		for _, e := range state {
			if e.State == Deleted {
				// We have already seen that this is deleted
				// so we do not need to publish again. The
				// user will be deleted from the store when
				// the state is closed.
				continue
			}
			if found[e.ID] {
				// We have the user, so we do not need to
				// mark it as deleted.
				continue
			}
			// This modifies the state store's copy since u
			// is a pointer held by the state store map.
			e.State = Deleted
			e.WhenChanged = now
			deleted[e.ID] = e
		}
		for _, d := range deleted {
			entries = append(entries, d)
		}
	}
	return entries
}

// runIncrementalUpdate will run an incremental update. The process is similar
// to full synchronization, except only users which have changed (newly
// discovered, modified, or deleted) will be published.
func (p *adInput) runIncrementalUpdate(inputCtx v2.Context, store *kvstore.Store, last time.Time, client beat.Client) (time.Time, error) {
	p.logger.Debugf("Running incremental update...")

	state, err := newStateStore(store)
	if err != nil {
		return last, fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer func() { // If commit is successful, call to this close will be no-op.
		closeErr := state.close(false)
		if closeErr != nil {
			p.logger.Errorw("Error rolling back incremental update transaction", "error", closeErr)
		}
	}()

	var updatedUsers, updatedDevices []*User
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	if p.cfg.wantUsers() {
		updatedUsers, err = p.doFetchUsers(ctx, state, false)
		if err != nil {
			return last, err
		}
	}
	if p.cfg.wantDevices() {
		updatedDevices, err = p.doFetchDevices(ctx, state, false)
		if err != nil {
			return last, err
		}
	}

	if len(updatedUsers) != 0 || len(updatedDevices) != 0 {
		tracker := kvstore.NewTxTracker(ctx)
		for _, u := range updatedUsers {
			p.publishUser(u, state, inputCtx.ID, client, tracker)
		}
		for _, d := range updatedDevices {
			p.publishDevice(d, state, inputCtx.ID, client, tracker)
		}
		tracker.Wait()
	}

	if ctx.Err() != nil {
		return last, ctx.Err()
	}

	// state.whenChanged is modified by the call to doFetchUsers to be
	// the latest modification time for all of the users that have been
	// collected in that call.
	latest := state.whenChanged
	state.lastUpdate = latest
	if err = state.close(true); err != nil {
		return last, fmt.Errorf("unable to commit state: %w", err)
	}

	return latest, nil
}

// doFetchUsers handles fetching user identities from Active Directory. If
// fullSync is true, then any existing whenChanged will be ignored, forcing a
// full synchronization from Active Directory. The whenChanged time of state
// is modified to be the time stamp of the latest User.WhenChanged value.
// Returns a set of modified users by ID.
func (p *adInput) doFetchUsers(ctx context.Context, state *stateStore, fullSync bool) ([]*User, error) {
	var since time.Time
	if !fullSync {
		since = state.whenChanged
	}

	query := "(&(objectCategory=person)(objectClass=user))"
	if p.cfg.UserQuery != "" {
		query = p.cfg.UserQuery
	}
	entries, err := activedirectory.GetDetails(query, p.cfg.URL, p.cfg.User, p.cfg.Password, p.baseDN, since, p.cfg.UserAttrs, p.cfg.GrpAttrs, p.cfg.PagingSize, nil, p.tlsConfig)
	p.logger.Debugf("received %d users from API", len(entries))
	if err != nil {
		return nil, err
	}

	users := make([]*User, 0, len(entries))
	for _, u := range entries {
		users = append(users, state.storeUser(u))
		if u.WhenChanged.After(state.whenChanged) {
			state.whenChanged = u.WhenChanged
		}
	}
	p.logger.Debugf("processed %d users from API", len(users))
	return users, nil
}

// doFetchDevices handles fetching device identities from Active Directory. If
// fullSync is true, then any existing whenChanged will be ignored, forcing a
// full synchronization from Active Directory. The whenChanged time of state
// is modified to be the time stamp of the latest User.WhenChanged value.
// Returns a set of modified users by ID.
func (p *adInput) doFetchDevices(ctx context.Context, state *stateStore, fullSync bool) ([]*User, error) {
	var since time.Time
	if !fullSync {
		since = state.whenChanged
	}

	query := "(&(objectClass=computer)(objectClass=user))"
	if p.cfg.DeviceQuery != "" {
		query = p.cfg.DeviceQuery
	}
	entries, err := activedirectory.GetDetails(query, p.cfg.URL, p.cfg.User, p.cfg.Password, p.baseDN, since, p.cfg.UserAttrs, p.cfg.GrpAttrs, p.cfg.PagingSize, nil, p.tlsConfig)
	p.logger.Debugf("received %d devices from API", len(entries))
	if err != nil {
		return nil, err
	}

	devices := make([]*User, 0, len(entries))
	for _, d := range entries {
		devices = append(devices, state.storeDevice(d))
		if d.WhenChanged.After(state.whenChanged) {
			state.whenChanged = d.WhenChanged
		}
	}
	p.logger.Debugf("processed %d devices from API", len(devices))
	return devices, nil
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

	_, _ = userDoc.Put("activedirectory", u.Entry)
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

// publishDevices will publish a device document using the given beat.Client.
func (p *adInput) publishDevice(u *User, state *stateStore, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	userDoc := mapstr.M{}

	_, _ = userDoc.Put("activedirectory", u.Entry)
	_, _ = userDoc.Put("labels.identity_source", inputID)
	_, _ = userDoc.Put("device.id", u.ID)

	switch u.State {
	case Deleted:
		_, _ = userDoc.Put("event.action", "device-deleted")
	case Discovered:
		_, _ = userDoc.Put("event.action", "device-discovered")
	case Modified:
		_, _ = userDoc.Put("event.action", "device-modified")
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    userDoc,
		Private:   tracker,
	}
	tracker.Add()

	p.logger.Debugf("Publishing device %q", u.ID)

	client.Publish(event)
}
