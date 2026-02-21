// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"

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
	PublishActionResult(req map[string]interface{}, res map[string]interface{})
}

type queryResultPublisher interface {
	Publish(index, idValue, idFieldKey, responseID string, meta map[string]interface{}, hits []map[string]interface{}, ecsm ecs.Mapping, reqData interface{})
}

type scheduledResponsePublisher interface {
	PublishScheduledResponse(scheduleID, responseID string, startedAt, completedAt time.Time, resultCount int, scheduleExecutionCount int64)
}

type scheduledQueryPublisher interface {
	queryResultPublisher
	scheduledResponsePublisher
}

type queryExecutor interface {
	Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]interface{}, error)
}

type namespaceProvider interface {
	GetNamespace() string
}

type actionHandler struct {
	log       *logp.Logger
	inputType string
	publisher queryResultPublisher
	queryExec queryExecutor
	np        namespaceProvider
}

func (a *actionHandler) Name() string {
	return a.inputType
}

// Execute handles the action request.
func (a *actionHandler) Execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {

	start := time.Now().UTC()
	count, responseID, err := a.execute(ctx, req)
	end := time.Now().UTC()

	res := map[string]interface{}{
		"started_at":   start.Format(time.RFC3339Nano),
		"completed_at": end.Format(time.RFC3339Nano),
		"response_id":  responseID,
	}

	if err != nil {
		res["error"] = err.Error()
	} else {
		res["count"] = count
	}
	return res, nil
}

func (a *actionHandler) execute(ctx context.Context, req map[string]interface{}) (int, string, error) {
	ac, err := action.FromMap(req)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %w", err, ErrQueryExecution)
	}
	responseID := uuid.Must(uuid.NewV4()).String()

	var namespace string
	if a.np != nil {
		namespace = a.np.GetNamespace()
	}
	if namespace == "" {
		namespace = config.DefaultNamespace
	}

	count, err := a.executeQuery(ctx, config.Datastream(namespace), ac, responseID, req)
	if err != nil {
		return 0, responseID, err
	}
	return count, responseID, nil
}

func (a *actionHandler) executeQuery(ctx context.Context, index string, ac action.Action, responseID string, req map[string]interface{}) (int, error) {

	if a.queryExec == nil {
		return 0, ErrNoQueryExecutor
	}
	if a.publisher == nil {
		return 0, ErrNoPublisher
	}

	a.log.Debugf("Execute query: %s", ac.Query)

	start := time.Now()

	hits, err := a.queryExec.Query(ctx, ac.Query, ac.Timeout)

	if err != nil {
		a.log.Errorf("Failed to execute query, err: %v", err)
		return 0, err
	}

	a.log.Debugf("Completed query in: %v", time.Since(start))

	a.publisher.Publish(index, ac.ID, "action_id", responseID, nil, hits, ac.ECSMapping, req["data"])

	return len(hits), nil
}
