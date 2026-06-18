// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/scheduler"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	defaultRRuleQueryTimeout  = 1 * time.Minute
	rruleRuntimeProfileSource = "rrule"
)

// rruleRuntimeProfileState holds optional live-style profiling for one RRULE execution.
type rruleRuntimeProfileState struct {
	queryName            string
	ns                   string
	responseID           string
	sql                  string
	shouldPublishProfile bool
	shouldCollectProfile bool
	before               runtimeSnapshot
	beforeReady          bool
}

// recurrenceQueryHandler handles RRULE-scheduled query execution
type recurrenceQueryHandler struct {
	log          *logp.Logger
	scheduler    *scheduler.Scheduler
	cli          *osqdcli.Client
	configPlugin *ConfigPlugin
	publisher    scheduledQueryPublisher
	profiles     liveProfileRecorder
	osqueryVer   string

	mx              sync.Mutex
	previousResults map[string]map[string]map[string]interface{}
}

// newRecurrenceQueryHandler creates a new RRULE query handler.
// profiles may be nil; when set, runtime-style profiles are recorded for every RRULE execution
// (same backing store as live queries), and policy-driven publish still follows LookupQueryProfile.
func newRecurrenceQueryHandler(log *logp.Logger, cli *osqdcli.Client, configPlugin *ConfigPlugin, pub scheduledQueryPublisher, profiles liveProfileRecorder, osqueryVersion string) *recurrenceQueryHandler {
	h := &recurrenceQueryHandler{
		log:             log.With("component", "rrule-query-handler"),
		cli:             cli,
		configPlugin:    configPlugin,
		publisher:       pub,
		profiles:        profiles,
		osqueryVer:      osqueryVersion,
		previousResults: make(map[string]map[string]map[string]interface{}),
	}

	h.scheduler = scheduler.New(log, h.executeQuery)
	return h
}

// Start starts the RRULE scheduler
func (h *recurrenceQueryHandler) Start(ctx context.Context) {
	h.scheduler.Start(ctx)
}

// Stop stops the RRULE scheduler
func (h *recurrenceQueryHandler) Stop() {
	h.scheduler.Stop()
}

// UpdateFromConfig updates RRULE-scheduled queries from osquery configuration
func (h *recurrenceQueryHandler) UpdateFromConfig(osqConfig *config.OsqueryConfig) error {
	if osqConfig == nil {
		return h.scheduler.UpdateQueries(nil)
	}

	var queries []*scheduler.ScheduledQuery

	// Process scheduled queries
	for name, q := range osqConfig.Schedule {
		if !q.RRuleSchedule.IsEnabled() {
			continue
		}

		sq, err := h.createScheduledQuery(name, q)
		if err != nil {
			return fmt.Errorf("osquery.schedule[%q]: %w", name, err)
		}
		if sq == nil {
			continue
		}
		queries = append(queries, sq)
	}

	// Process packs
	for packName, pack := range osqConfig.Packs {
		for queryName, q := range pack.Queries {
			// Expect merged pack defaults (callers should pass ConfigPlugin.EffectiveOsqueryConfig
			// after a successful Set, not raw agent input).
			if !q.RRuleSchedule.IsEnabled() {
				continue
			}

			fullName := getPackQueryName(packName, queryName)
			sq, err := h.createScheduledQuery(fullName, q)
			if err != nil {
				return fmt.Errorf("osquery.packs[%q].queries[%q]: %w", packName, queryName, err)
			}
			if sq == nil {
				continue
			}
			queries = append(queries, sq)
		}
	}

	if err := h.scheduler.UpdateQueries(queries); err != nil {
		return err
	}
	h.retainDiffState(queries)
	return nil
}

// createScheduledQuery creates a ScheduledQuery from config
func (h *recurrenceQueryHandler) createScheduledQuery(name string, q config.Query) (*scheduler.ScheduledQuery, error) {
	rruleConfig := q.RRuleSchedule
	if !platformMatches(q.Platform, runtime.GOOS) {
		h.log.Debugf("Skipping RRULE-scheduled query '%s': platform %q does not match %q", name, q.Platform, runtime.GOOS)
		return nil, nil
	}
	if !versionMatches(q.Version, h.osqueryVer) {
		h.log.Debugf("Skipping RRULE-scheduled query '%s': requires osquery >= %q, current %q", name, q.Version, h.osqueryVer)
		return nil, nil
	}

	startDate, err := rruleConfig.ParseStartDate()
	if err != nil {
		return nil, err
	}

	endDate, err := rruleConfig.ParseEndDate()
	if err != nil {
		return nil, err
	}

	// Parse splay duration
	splay, err := rruleConfig.GetSplay()
	if err != nil {
		return nil, err
	}

	recurrenceSchedule := &scheduler.RecurrenceSchedule{
		RRule:     rruleConfig.RRule,
		StartDate: startDate,
		EndDate:   endDate,
		Splay:     splay,
	}

	if err := recurrenceSchedule.Parse(); err != nil {
		return nil, err
	}

	timeout := defaultRRuleQueryTimeout
	if rruleConfig.Timeout > 0 {
		timeout = time.Duration(rruleConfig.Timeout) * time.Second
	}

	return &scheduler.ScheduledQuery{
		Name:     name,
		Config:   q,
		Timeout:  timeout,
		Schedule: recurrenceSchedule,
	}, nil
}

func platformMatches(selector, goos string) bool {
	selector = strings.TrimSpace(strings.ToLower(selector))
	if selector == "" || selector == "all" {
		return true
	}
	for _, platform := range strings.Split(selector, ",") {
		switch strings.TrimSpace(strings.ToLower(platform)) {
		case "all":
			return true
		case goos:
			return true
		case "posix":
			if goos == "linux" || goos == "darwin" {
				return true
			}
		}
	}
	return false
}

func versionMatches(minVersion, currentVersion string) bool {
	minVersion = strings.TrimSpace(minVersion)
	if minVersion == "" {
		return true
	}
	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion == "" {
		return false
	}
	min, err := semver.NewVersion(minVersion)
	if err != nil {
		return false
	}
	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false
	}
	return !current.LessThan(min)
}

func (h *recurrenceQueryHandler) retainDiffState(queries []*scheduler.ScheduledQuery) {
	retain := make(map[string]struct{}, len(queries))
	for _, q := range queries {
		if q != nil {
			retain[q.Name] = struct{}{}
		}
	}

	h.mx.Lock()
	defer h.mx.Unlock()
	for name := range h.previousResults {
		if _, ok := retain[name]; !ok {
			delete(h.previousResults, name)
		}
	}
}

// initRRuleRuntimeProfiling collects a pre-query process snapshot when profiling is enabled
// for this query (publish flag and/or local profile store).
func (h *recurrenceQueryHandler) initRRuleRuntimeProfiling(ctx context.Context, name, ns, responseID, sql string) rruleRuntimeProfileState {
	st := rruleRuntimeProfileState{
		queryName:            name,
		ns:                   ns,
		responseID:           responseID,
		sql:                  sql,
		shouldPublishProfile: h.configPlugin.LookupQueryProfile(name),
	}
	st.shouldCollectProfile = st.shouldPublishProfile || h.profiles != nil
	if !st.shouldCollectProfile {
		return st
	}
	snapshot, snapErr := collectRuntimeSnapshot(ctx, h.cli)
	if snapErr != nil {
		h.log.Debugf("failed to collect pre-query profile snapshot for %s: %v", name, snapErr)
		return st
	}
	st.before = snapshot
	st.beforeReady = true
	return st
}

// completeRRuleRuntimeProfiling collects the post-query snapshot and records or publishes
// the profile. queryErr is the error from the user query, if any.
func (h *recurrenceQueryHandler) completeRRuleRuntimeProfiling(ctx context.Context, st rruleRuntimeProfileState, queryDuration time.Duration, queryErr error) {
	if !st.shouldCollectProfile {
		return
	}
	if !st.beforeReady {
		if st.shouldPublishProfile {
			h.log.Debug("profile requested but skipped: pre-query snapshot was not collected")
		} else {
			h.log.Debug("profile storage skipped: pre-query snapshot was not collected")
		}
		return
	}
	after, snapErr := collectRuntimeSnapshot(ctx, h.cli)
	if snapErr != nil {
		h.log.Debugf("failed to collect post-query profile snapshot for %s: %v", st.queryName, snapErr)
		return
	}
	prof := buildRuntimeQueryProfile(rruleRuntimeProfileSource, st.sql, st.before, after, queryDuration, queryErr)
	if h.profiles != nil {
		h.profiles.RecordLiveProfile(st.sql, prof)
	}
	if st.shouldPublishProfile {
		h.publisher.PublishQueryProfile(config.QueryProfileDatastream(st.ns), st.queryName, "", st.responseID, prof, nil)
	}
}

// executeQuery is called by the scheduler to execute a query
func (h *recurrenceQueryHandler) executeQuery(ctx context.Context, scheduledQuery scheduler.ScheduledQuery, executionIndex int, plannedScheduleTime time.Time) error {
	name := scheduledQuery.Name
	query := scheduledQuery.SQL()
	h.log.Debugf("Executing RRULE-scheduled query '%s' (execution #%d)", name, executionIndex)

	ns, ok := h.configPlugin.LookupNamespace(name)
	if !ok {
		ns = config.DefaultNamespace
	}

	responseID := uuid.Must(uuid.NewV4()).String()
	profSt := h.initRRuleRuntimeProfiling(ctx, name, ns, responseID, query)

	startedAt := time.Now()
	hits, err := h.cli.Query(ctx, query, scheduledQuery.Timeout)
	queryDuration := time.Since(startedAt)
	completedAt := time.Now()
	h.completeRRuleRuntimeProfiling(ctx, profSt, queryDuration, err)

	if err != nil {
		return err
	}

	// Get query info for ECS mapping, pack/space, and schedule id fallback
	var ecsMapping ecs.Mapping
	var spaceID, packID string
	if qi, ok := h.configPlugin.LookupQueryInfo(name); ok {
		ecsMapping = qi.ECSMapping
		spaceID = qi.SpaceID
		packID = qi.PackID
	}

	scheduleID := scheduledQuery.ScheduleID()

	baseMeta := map[string]interface{}{
		"unix_time":                completedAt.Unix(),
		"planned_schedule_time":    plannedScheduleTime.Format(time.RFC3339Nano),
		"rrule_query":              true,
		"scheduled_by":             "rrule",
		"schedule_execution_count": executionIndex,
	}

	totalHits := 0
	publish := func(typ, action string, rows []map[string]interface{}) {
		if len(rows) == 0 {
			return
		}
		totalHits += len(rows)
		meta := cloneRRuleMeta(baseMeta)
		meta["type"] = typ
		if action != "" {
			meta["action"] = action
		}
		h.publisher.Publish(config.Datastream(ns), scheduleID, "schedule_id", responseID, spaceID, packID, meta, rows, ecsMapping, nil)
	}

	if scheduledQuery.Snapshot() {
		publish("snapshot", "", hits)
	} else {
		added, removed := h.diffResults(name, hits, scheduledQuery.Removed())
		publish("diff", "added", added)
		publish("diff", "removed", removed)
	}

	// Synthetic response document (no action) with execution count for correlation
	h.publisher.PublishScheduledResponse(scheduleID, packID, spaceID, responseID, startedAt, completedAt, plannedScheduleTime, totalHits, int64(executionIndex))

	h.log.Debugf("RRULE-scheduled query '%s' completed with %d results", name, len(hits))
	return nil
}

func cloneRRuleMeta(meta map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(meta)+2)
	for k, v := range meta {
		out[k] = v
	}
	return out
}

func (h *recurrenceQueryHandler) diffResults(name string, hits []map[string]interface{}, includeRemoved bool) ([]map[string]interface{}, []map[string]interface{}) {
	current := rowsByKey(hits)

	h.mx.Lock()
	previous := h.previousResults[name]
	h.previousResults[name] = current
	h.mx.Unlock()

	added, removed := diffRows(previous, current, includeRemoved)
	return added, removed
}

func diffRows(previous, current map[string]map[string]interface{}, includeRemoved bool) ([]map[string]interface{}, []map[string]interface{}) {
	var added []map[string]interface{}
	for key, row := range current {
		if _, ok := previous[key]; !ok {
			added = append(added, row)
		}
	}
	if !includeRemoved {
		return added, nil
	}
	var removed []map[string]interface{}
	for key, row := range previous {
		if _, ok := current[key]; !ok {
			removed = append(removed, row)
		}
	}
	return added, removed
}

func rowsByKey(rows []map[string]interface{}) map[string]map[string]interface{} {
	out := make(map[string]map[string]interface{}, len(rows))
	for _, row := range rows {
		out[rowKey(row)] = row
	}
	return out
}

func rowKey(row map[string]interface{}) string {
	raw, err := json.Marshal(row)
	if err != nil {
		return fmt.Sprintf("%v", row)
	}
	return string(raw)
}
