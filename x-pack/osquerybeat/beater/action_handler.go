// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
)

type actionHandler struct {
	log       *logp.Logger
	inputType string
	bt        *osquerybeat
	cli       *osqdcli.Client
}

func (a *actionHandler) Name() string {
	return a.inputType
}

type actionData struct {
	Query string
	ID    string
}

func actionDataFromRequest(req map[string]interface{}) (ad actionData, err error) {
	if len(req) == 0 {
		return ad, ErrActionRequest
	}
	if v, ok := req["id"]; ok {
		if id, ok := v.(string); ok {
			ad.ID = id
		}
	}
	if v, ok := req["data"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			if v, ok := m["query"]; ok {
				if query, ok := v.(string); ok {
					ad.Query = query
				}
			}
		}
	}
	return ad, nil
}

// Execute handles the action request.
func (a *actionHandler) Execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {

	start := time.Now().UTC()
	err := a.execute(ctx, req)
	end := time.Now().UTC()

	res := map[string]interface{}{
		"started_at":   start.Format(time.RFC3339Nano),
		"completed_at": end.Format(time.RFC3339Nano),
	}

	if err != nil {
		res["error"] = err.Error()
	}
	return res, nil
}

func (a *actionHandler) execute(ctx context.Context, req map[string]interface{}) error {
	ad, err := actionDataFromRequest(req)
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrQueryExecution)
	}
	return a.executeQuery(ctx, config.DefaultStreamIndex, ad.ID, ad.Query, "", req)
}

func (a *actionHandler) executeQuery(ctx context.Context, index, id, query, responseID string, req map[string]interface{}) error {

	a.log.Debugf("Execute query: %s", query)

	start := time.Now()

	hits, err := a.cli.Query(ctx, query)

	if err != nil {
		a.log.Errorf("Failed to execute query, err: %v", err)
		return err
	}

	a.log.Debugf("Completed query in: %v", time.Since(start))

	if err != nil {
		return err
	}
	a.bt.publishEvents(index, id, responseID, hits, req["data"])
	return nil
}
