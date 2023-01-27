// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package azuread provides an identity asset provider for Azure Active Directory.
package azuread

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator/oauth2"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher/graph"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"
)

// Name of this provider.
const Name = "azure-ad"

// FullName of this provider, including the input name. Prefer using this
// value for full context, especially if the input name isn't present in an
// adjacent log field.
const FullName = "entity-analytics-" + Name

var _ provider.Provider = &azure{}

// azure implements the provider.Provider interface.
type azure struct {
	*kvstore.Manager

	conf conf

	metrics *inputMetrics
	logger  *logp.Logger
	auth    authenticator.Authenticator
	fetcher fetcher.Fetcher
}

// Name returns the name of this provider.
func (p *azure) Name() string {
	return FullName
}

// Test will test the provider by verifying an OAuth2 token can be obtained.
func (p *azure) Test(testCtx v2.TestContext) error {
	p.logger = testCtx.Logger.With("tenant_id", p.conf.TenantID, "provider", Name)
	p.auth.SetLogger(p.logger)

	ctx := ctxtool.FromCanceller(testCtx.Cancelation)
	if _, err := p.auth.Token(ctx); err != nil {
		return fmt.Errorf("%s test failed: %w", Name, err)
	}

	return nil
}

// Run will start data collection on this provider.
func (p *azure) Run(inputCtx v2.Context, store *kvstore.Store, client beat.Client) error {
	p.logger = inputCtx.Logger.With("tenant_id", p.conf.TenantID, "provider", Name)
	p.auth.SetLogger(p.logger)
	p.fetcher.SetLogger(p.logger)
	p.metrics = newMetrics(inputCtx.ID, nil)
	defer p.metrics.Close()

	lastSyncTime, _ := getLastSync(store)
	syncWaitTime := time.Until(lastSyncTime.Add(p.conf.SyncInterval))
	lastUpdateTime, _ := getLastUpdate(store)
	updateWaitTime := time.Until(lastUpdateTime.Add(p.conf.UpdateInterval))

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
				p.logger.Errorf("Error running full sync: %v", err)
				p.metrics.syncError.Inc()
			}
			p.metrics.syncTotal.Inc()
			p.metrics.syncProcessingTime.Update(time.Since(start).Nanoseconds())

			syncTimer.Reset(p.conf.SyncInterval)
			p.logger.Debugf("Next sync expected at: %v", time.Now().Add(p.conf.SyncInterval))

			// Reset the update timer and wait the configured interval. If the
			// update timer has already fired, then drain the timer's channel
			// before resetting.
			if !updateTimer.Stop() {
				<-updateTimer.C
			}
			updateTimer.Reset(p.conf.UpdateInterval)
			p.logger.Debugf("Next update expected at: %v", time.Now().Add(p.conf.UpdateInterval))
		case <-updateTimer.C:
			start := time.Now()
			if err := p.runIncrementalUpdate(inputCtx, store, client); err != nil {
				p.logger.Errorf("Error running incremental update: %v", err)
				p.metrics.updateError.Inc()
			}
			p.metrics.updateTotal.Inc()
			p.metrics.updateProcessingTime.Update(time.Since(start).Nanoseconds())
			updateTimer.Reset(p.conf.UpdateInterval)
			p.logger.Debugf("Next update expected at: %v", time.Now().Add(p.conf.UpdateInterval))
		}
	}
}

// runFullSync performs a full synchronization. It will fetch user and group
// identities from Azure Active Directory, enrich users with group memberships,
// and publishes all known users (regardless if they have been modified) to the
// given beat.Client.
func (p *azure) runFullSync(inputCtx v2.Context, store *kvstore.Store, client beat.Client) (err error) {
	p.logger.Debugf("Running full sync...")

	p.logger.Debugf("Opening new transaction...")
	state, err := newStateStore(store)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	p.logger.Debugf("Transaction opened")
	defer func() { // If commit is successful, call to this close will be no-op.
		if err = state.close(false); err != nil {
			p.logger.Errorf("Error rolling back transaction: %v", err)
		}
	}()

	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	p.logger.Debugf("Starting fetch...")
	if _, err := p.doFetch(ctx, state, true); err != nil {
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
	if err = state.close(true); err != nil {
		return fmt.Errorf("unable to commit state: %w", err)
	}

	return nil
}

// runIncrementalUpdate will run an incremental update. The process is similar
// to full synchronization, except only users which have changed (newly
// discovered, modified, or deleted) will be published.
func (p *azure) runIncrementalUpdate(inputCtx v2.Context, store *kvstore.Store, client beat.Client) (err error) {
	p.logger.Debugf("Running incremental update...")

	state, err := newStateStore(store)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer func() { // If commit is successful, call to this close will be no-op.
		if err = state.close(false); err != nil {
			p.logger.Errorf("Error rolling back transaction: %v", err)
		}
	}()

	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	updatedUsers, err := p.doFetch(ctx, state, false)
	if err != nil {
		return err
	}

	if updatedUsers.Len() != 0 {
		tracker := kvstore.NewTxTracker(ctx)
		updatedUsers.ForEach(func(id uuid.UUID) {
			u, ok := state.users[id]
			if !ok {
				p.logger.Warnf("Unable to lookup user %q", id, u.ID)
				return
			}
			p.publishUser(u, state, inputCtx.ID, client, tracker)
		})

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

// doFetch handles fetching user and group identities from Azure Active Directory
// and enriching users with group memberships. If fullSync is true, then any
// existing deltaLink will be ignored, forcing a full synchronization from
// Azure Active Directory. Returns a set of modified users by ID.
func (p *azure) doFetch(ctx context.Context, state *stateStore, fullSync bool) (collections.UUIDSet, error) {
	var updatedUsers collections.UUIDSet
	var usersDeltaLink string
	var groupsDeltaLink string

	// Get user changes.
	if !fullSync {
		usersDeltaLink = state.usersLink
		groupsDeltaLink = state.groupsLink
	}

	changedUsers, userLink, err := p.fetcher.Users(ctx, usersDeltaLink)
	if err != nil {
		return updatedUsers, err
	}
	p.logger.Debugf("Received %d users from API", len(changedUsers))

	// Get group changes.
	changedGroups, groupLink, err := p.fetcher.Groups(ctx, groupsDeltaLink)
	if err != nil {
		return updatedUsers, err
	}
	p.logger.Debugf("Received %d groups from API", len(changedGroups))

	state.usersLink = userLink
	state.groupsLink = groupLink

	for _, v := range changedUsers {
		updatedUsers.Add(v.ID)
		state.storeUser(v)
	}
	for _, v := range changedGroups {
		state.storeGroup(v)
	}

	// Populate group relationships tree.
	for _, g := range changedGroups {
		if g.Deleted {
			for _, u := range state.users {
				if u.TransitiveMemberOf.Contains(g.ID) {
					updatedUsers.Add(u.ID)
				}
			}
			state.relationships.RemoveVertex(g.ID)
			continue
		}

		for _, member := range g.Members {
			switch member.Type {
			case fetcher.MemberGroup:
				for _, u := range state.users {
					if u.TransitiveMemberOf.Contains(member.ID) {
						updatedUsers.Add(u.ID)
					}
				}
				if member.Deleted {
					state.relationships.RemoveEdge(member.ID, g.ID)
				} else {
					state.relationships.AddEdge(member.ID, g.ID)
				}

			case fetcher.MemberUser:
				if u, ok := state.users[member.ID]; ok {
					updatedUsers.Add(u.ID)
					if member.Deleted {
						u.MemberOf.Remove(g.ID)
					} else {
						u.MemberOf.Add(g.ID)
					}
				}
			}
		}
	}

	// Expand user group memberships.
	updatedUsers.ForEach(func(userID uuid.UUID) {
		u, ok := state.users[userID]
		if !ok {
			p.logger.Errorf("Unable to find user %q in state", userID)
			return
		}
		u.Modified = true
		if u.Deleted {
			return
		}

		u.TransitiveMemberOf = *state.relationships.ExpandFromSet(u.MemberOf)
	})

	return updatedUsers, nil
}

// publishMarker will publish a write marker document using the given beat.Client.
// If start is true, then it will be a start marker, otherwise an end marker.
func (p *azure) publishMarker(ts, eventTime time.Time, inputID string, start bool, client beat.Client, tracker *kvstore.TxTracker) {
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
func (p *azure) publishUser(u *fetcher.User, state *stateStore, inputID string, client beat.Client, tracker *kvstore.TxTracker) {
	userDoc := mapstr.M{}

	_, _ = userDoc.Put("azure_ad", u.Fields)
	_, _ = userDoc.Put("labels.identity_source", inputID)
	_, _ = userDoc.Put("user.id", u.ID.String())

	if u.Deleted {
		_, _ = userDoc.Put("event.action", "user-deleted")
	} else if u.Discovered {
		_, _ = userDoc.Put("event.action", "user-discovered")
	} else if u.Modified {
		_, _ = userDoc.Put("event.action", "user-modified")
	}

	var groups []fetcher.GroupECS
	u.TransitiveMemberOf.ForEach(func(groupID uuid.UUID) {
		g, ok := state.groups[groupID]
		if !ok {
			p.logger.Warnf("Unable to lookup group %q for user %q", groupID, u.ID)
			return
		}
		groups = append(groups, g.ToECS())
	})
	if len(groups) != 0 {
		_, _ = userDoc.Put("user.group", groups)
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

// configure configures this provider using the given configuration.
func (p *azure) configure(cfg *config.C) (kvstore.Input, error) {
	var err error

	if err = cfg.Unpack(&p.conf); err != nil {
		return nil, fmt.Errorf("unable to unpack %s input config: %w", Name, err)
	}

	if p.auth, err = oauth2.New(cfg, p.Manager.Logger); err != nil {
		return nil, fmt.Errorf("unable to create authenticator: %w", err)
	}
	if p.fetcher, err = graph.New(cfg, p.Manager.Logger, p.auth); err != nil {
		return nil, fmt.Errorf("unable to create fetcher: %w", err)
	}

	return p, nil
}

// New creates a new instance of an Azure Active Directory identity provider.
func New(logger *logp.Logger) (provider.Provider, error) {
	p := azure{
		conf: defaultConf(),
	}
	p.Manager = &kvstore.Manager{
		Logger:    logger,
		Type:      FullName,
		Configure: p.configure,
	}

	return &p, nil
}

func init() {
	if err := provider.Register(Name, New); err != nil {
		panic(err)
	}
}
