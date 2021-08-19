// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
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

type actionData struct {
	Query      string
	ID         string
	ECSMapping ecs.Mapping
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
			if v, ok := m["ecs_mapping"]; ok {
				m, ok := v.(map[string]interface{})
				if !ok {
					return ad, ErrActionRequest
				}
				ecsm, err := convertActionDataECSMapping(m)
				if err != nil {
					return ad, err
				}
				ad.ECSMapping = ecsm
			}
		}
	}
	return ad, nil
}

func convertActionDataECSMapping(m map[string]interface{}) (ecsm ecs.Mapping, err error) {
	ecsm = make(ecs.Mapping)
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			return ecsm, ErrActionRequest
		}
		valmap, ok := v.(map[string]interface{})
		if !ok {
			return ecsm, ErrActionRequest
		}

		var (
			val   interface{}
			field string
		)

		if val, ok = valmap["field"]; ok {
			// "field" can only be string
			field, ok = val.(string)
			if !ok {
				return ecsm, ErrActionRequest
			}
			field = strings.TrimSpace(field)
		}

		value := valmap["value"]
		if field != "" && value != nil {
			return ecsm, ErrActionRequest
		}

		// Should have at field or value defined in the mapping object
		if field == "" && value == nil {
			return ecsm, ErrActionRequest
		}
		ecsm[k] = ecs.MappingInfo{
			Field: field,
			Value: value,
		}
	}
	return
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
	return a.executeQuery(ctx, config.DefaultStreamIndex, ad, "", req)
}

func (a *actionHandler) executeQuery(ctx context.Context, index string, ad actionData, responseID string, req map[string]interface{}) error {

	a.log.Debugf("Execute query: %s", ad.Query)

	start := time.Now()

	hits, err := a.cli.Query(ctx, ad.Query)

	if err != nil {
		a.log.Errorf("Failed to execute query, err: %v", err)
		return err
	}

	a.log.Debugf("Completed query in: %v", time.Since(start))

	var ecsFields []common.MapStr
	// If non-empty result and the ECSMapping is present
	if len(hits) > 0 && len(ad.ECSMapping) > 0 {
		ecsFields = make([]common.MapStr, len(hits))
		for i, hit := range hits {
			ecsFields[i] = common.MapStr(ad.ECSMapping.Map(ecs.Doc(hit)))
		}
	}
	if err != nil {
		return err
	}
	a.bt.publishEvents(index, ad.ID, responseID, hits, ecsFields, req["data"])
	return nil
}
