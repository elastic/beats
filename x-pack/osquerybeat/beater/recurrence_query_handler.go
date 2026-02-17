// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/scheduler"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	defaultRRuleQueryTimeout = 1 * time.Minute
)

// recurrenceQueryHandler handles RRULE-scheduled query execution
type recurrenceQueryHandler struct {
	log          *logp.Logger
	scheduler    *scheduler.Scheduler
	cli          *osqdcli.Client
	configPlugin *ConfigPlugin
	publisher    scheduledQueryPublisher
}

// newRecurrenceQueryHandler creates a new RRULE query handler
func newRecurrenceQueryHandler(log *logp.Logger, cli *osqdcli.Client, configPlugin *ConfigPlugin, pub scheduledQueryPublisher) *recurrenceQueryHandler {
	h := &recurrenceQueryHandler{
		log:          log.With("component", "rrule-query-handler"),
		cli:          cli,
		configPlugin: configPlugin,
		publisher:    pub,
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
			h.log.Errorf("Failed to create scheduled query '%s': %v", name, err)
			continue
		}
		queries = append(queries, sq)
	}

	// Process packs
	for packName, pack := range osqConfig.Packs {
		for queryName, q := range pack.Queries {
			// Use query-level rrule schedule if set, otherwise fall back to pack-level
			rruleConfig := q.RRuleSchedule
			if rruleConfig == nil || !rruleConfig.IsEnabled() {
				rruleConfig = pack.RRuleSchedule
			}
			if rruleConfig == nil || !rruleConfig.IsEnabled() {
				continue
			}

			// Create a copy with the effective rrule config
			queryCopy := q
			queryCopy.RRuleSchedule = rruleConfig

			fullName := getPackQueryName(packName, queryName)
			sq, err := h.createScheduledQuery(fullName, queryCopy)
			if err != nil {
				h.log.Errorf("Failed to create scheduled query '%s': %v", fullName, err)
				continue
			}
			queries = append(queries, sq)
		}
	}

	return h.scheduler.UpdateQueries(queries)
}

// createScheduledQuery creates a ScheduledQuery from config
func (h *recurrenceQueryHandler) createScheduledQuery(name string, q config.Query) (*scheduler.ScheduledQuery, error) {
	rruleConfig := q.RRuleSchedule

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
		Query:    q.Query,
		Timeout:  timeout,
		Schedule: recurrenceSchedule,
		ScheduleID: q.ScheduleID,
	}, nil
}

// executeQuery is called by the scheduler to execute a query
func (h *recurrenceQueryHandler) executeQuery(ctx context.Context, name, query string, timeout time.Duration, scheduleID string, executionIndex int) error {
	h.log.Debugf("Executing RRULE-scheduled query '%s' (execution #%d)", name, executionIndex)

	startedAt := time.Now()

	// Execute the query via osquery client
	// Note: Query() already resolves the result types
	hits, err := h.cli.Query(ctx, query, timeout)
	if err != nil {
		return err
	}

	completedAt := time.Now()

	// Get namespace for this query
	ns, ok := h.configPlugin.LookupNamespace(name)
	if !ok {
		ns = config.DefaultNamespace
	}

	// Get query info for ECS mapping
	var ecsMapping ecs.Mapping
	if qi, ok := h.configPlugin.LookupQueryInfo(name); ok {
		ecsMapping = qi.ECSMapping
	}

	// Use policy schedule_id when provided
	if scheduleID == "" {
		scheduleID = name
	}

	// Generate a response ID
	responseID := uuid.Must(uuid.NewV4()).String()

	// Publish results with schedule_execution_count (from RRULE + start_date)
	meta := map[string]interface{}{
		"type":                     "rrule_snapshot",
		"unix_time":                completedAt.Unix(),
		"rrule_query":              true,
		"scheduled_by":             "rrule",
		"schedule_execution_count": executionIndex,
	}

	h.publisher.Publish(config.Datastream(ns), scheduleID, "schedule_id", responseID, meta, hits, ecsMapping, nil)

	// Synthetic response document (no action) with execution count for correlation
	h.publisher.PublishScheduledResponse(scheduleID, responseID, startedAt, completedAt, len(hits), int64(executionIndex))

	h.log.Debugf("RRULE-scheduled query '%s' completed with %d results", name, len(hits))
	return nil
}
