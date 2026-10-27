// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/action"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	ErrNoPublisher     = errors.New("no publisher configured")
	ErrNoQueryExecutor = errors.New("no query executor configures")
)

type actionResultPublisher interface {
	PublishActionResult(req map[string]any, res map[string]any)
}

type queryResultPublisher interface {
	Publish(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any)
}

type scheduledResponsePublisher interface {
	PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID string, startedAt, completedAt, plannedScheduleTime time.Time, resultCount int, scheduleExecutionCount int64)
}

type queryProfilePublisher interface {
	// PublishQueryProfile publishes a live query's profile and reports
	// whether it was handed to the output: false when no profile stream is
	// configured or ctx cancelled the publish.
	PublishQueryProfile(ctx context.Context, index, queryName, actionID, responseID, spaceID string, profile map[string]any, reqData any) bool
}

type scheduledProfilePublisher interface {
	PublishScheduledProfile(index, queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any)
}

type liveProfileRecorder interface {
	RecordLiveProfile(query string, profile map[string]any)
}

type scheduledQueryPublisher interface {
	queryResultPublisher
	scheduledResponsePublisher
	scheduledProfilePublisher
}

type actionQueryPublisher interface {
	// PublishEach publishes hits one by one and returns how many were handed
	// to the output before stop returned true or ctx ended. stop is called
	// before each hit.
	PublishEach(ctx context.Context, index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any, stop func() bool) int
	queryProfilePublisher
}

type queryExecutor interface {
	Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]any, error)
}

type namespaceProvider interface {
	GetNamespace() string
}

// profileDefaultsProvider exposes the fleet-wide profiling default so live (ad-hoc)
// queries can resolve it against their optional per-action override.
type profileDefaultsProvider interface {
	GlobalProfileEnabled() bool
}

type actionHandler struct {
	log             *logp.Logger
	inputType       string
	publisher       actionQueryPublisher
	live            *liveResultDispatcher
	queryExec       queryExecutor
	np              namespaceProvider
	profiles        liveProfileRecorder
	profileDefaults profileDefaultsProvider
}

// profileEnabled resolves whether this action should publish a profile, combining the
// global default with the optional per-action override.
func (a *actionHandler) profileEnabled(override *bool) bool {
	var global bool
	if a.profileDefaults != nil {
		global = a.profileDefaults.GlobalProfileEnabled()
	}
	return config.ResolveProfiling(global, override)
}

func (a *actionHandler) Name() string {
	return a.inputType
}

// Execute handles the action request.
func (a *actionHandler) Execute(ctx context.Context, req map[string]any) (map[string]any, error) {

	start := time.Now().UTC()
	count, err := a.execute(ctx, req)
	end := time.Now().UTC()

	res := map[string]any{
		"started_at":   start.Format(time.RFC3339Nano),
		"completed_at": end.Format(time.RFC3339Nano),
	}

	if err != nil {
		res["error"] = err.Error()
	}
	// A publish timeout reports the rows handed to the output too.
	if err == nil || errors.Is(err, errLivePublishTimeout) {
		res["count"] = count
	}
	return res, nil
}

func (a *actionHandler) execute(ctx context.Context, req map[string]any) (int, error) {
	ac, err := action.FromMap(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", err, ErrQueryExecution)
	}
	if !ac.MatchesPlatform() {
		a.log.Debugf("Skipping query for platforms: %v", ac.Platforms)
		return 0, nil
	}

	return a.executeQuery(ctx, config.Datastream(a.namespace()), ac, "", req)
}

func (a *actionHandler) namespace() string {
	if a.np != nil {
		if ns := a.np.GetNamespace(); ns != "" {
			return ns
		}
	}
	return config.DefaultNamespace
}

func (a *actionHandler) executeQuery(ctx context.Context, index string, ac action.Action, responseID string, req map[string]any) (int, error) {

	if a.queryExec == nil {
		return 0, ErrNoQueryExecutor
	}
	if a.publisher == nil {
		return 0, ErrNoPublisher
	}

	a.log.Debugf("Execute query: %s", ac.Query)

	job := liveJob{index: index, actionID: ac.ID, responseID: responseID, spaceID: ac.SpaceID, ecsm: ac.ECSMapping, reqData: req["data"]}
	var before runtimeSnapshot
	beforeReady := false
	publishProfile := a.profileEnabled(ac.Profile)
	shouldCollect := publishProfile || a.profiles != nil
	if shouldCollect {
		snapshot, err := collectRuntimeSnapshot(ctx, a.queryExec)
		if err != nil {
			a.log.Debugf("failed to collect pre-query profile snapshot: %v", err)
		} else {
			before = snapshot
			beforeReady = true
		}
	}

	start := time.Now()

	hits, err := a.queryExec.Query(ctx, ac.Query, ac.Timeout)
	duration := time.Since(start)

	if shouldCollect && beforeReady {
		after, snapErr := collectRuntimeSnapshot(ctx, a.queryExec)
		if snapErr != nil {
			a.log.Debugf("failed to collect post-query profile snapshot: %v", snapErr)
		} else {
			profile := buildLiveQueryProfile(ac.Query, before, after, duration, err)
			if a.profiles != nil {
				a.profiles.RecordLiveProfile(ac.Query, profile)
			}
			if publishProfile {
				job.profileIndex, job.profile = config.QueryProfileDatastream(a.namespace()), profile
			}
		}
	} else if shouldCollect && !beforeReady {
		if publishProfile {
			a.log.Debug("profile requested but skipped: pre-query snapshot was not collected")
		} else {
			a.log.Debug("profile storage skipped: pre-query snapshot was not collected")
		}
	}

	if err != nil {
		a.log.Errorf("Failed to execute query, err: %v", err)
		if job.profile != nil {
			// The action reports the query error, not the profile's outcome.
			_, _ = a.live.publishResults(ctx, a.publisher, job)
		}
		return 0, err
	}

	a.log.Debugf("Completed query in: %v", duration)

	// The publish timeout starts now, so a slow query never reports partial.
	job.hits = hits
	return a.live.publishResults(ctx, a.publisher, job)
}
