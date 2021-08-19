// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/action"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
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
	ac, err := action.FromMap(req)
	if err != nil {
		return fmt.Errorf("%v: %w", err, ErrQueryExecution)
	}
	return a.executeQuery(ctx, config.Datastream(config.DefaultNamespace), ac, "", req)
}

func (a *actionHandler) executeQuery(ctx context.Context, index string, ac action.Action, responseID string, req map[string]interface{}) error {

	a.log.Debugf("Execute query: %s", ac.Query)

	start := time.Now()

	hits, err := a.cli.Query(ctx, ac.Query)

	if err != nil {
		a.log.Errorf("Failed to execute query, err: %v", err)
		return err
	}

	a.log.Debugf("Completed query in: %v", time.Since(start))

	var ecsFields []common.MapStr
	// If non-empty result and the ECSMapping is present
	if len(hits) > 0 && len(ac.ECSMapping) > 0 {
		ecsFields = make([]common.MapStr, len(hits))
		for i, hit := range hits {
			ecsFields[i] = common.MapStr(ac.ECSMapping.Map(ecs.Doc(hit)))
		}
	}
	if err != nil {
		return err
	}
	a.bt.publishEvents(index, ac.ID, responseID, hits, ecsFields, req["data"])
	return nil
}
