// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcpbigquery

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/cespare/xxhash/v2"
	"google.golang.org/api/option"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const inputName = "gcpbigquery"

var _ cursor.Source = (*bigQuerySource)(nil)
var _ cursor.Input = (*bigQueryInput)(nil)

func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  configure,
		},
	}
}

func configure(cfg *conf.C, logger *logp.Logger) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	var sources []cursor.Source
	for _, query := range config.Queries {
		source := bigQuerySource{
			ProjectID:      config.ProjectID,
			Query:          query.Query,
			TimestampField: query.TimestampField,
			IdFields:       query.IdFields,
			ExpandJson:     true,
		}

		if query.ExpandJsonStrings != nil {
			source.ExpandJson = *query.ExpandJsonStrings
		}

		if query.Cursor != nil {
			source.CursorField = query.Cursor.Field
			source.CursorInitialValue = query.Cursor.InitialValue
		}

		sources = append(sources, &source)
	}

	return sources, &bigQueryInput{config: config, logger: logger}, nil
}

func updateStatus(ctx v2.Context, status status.Status, msg string) {
	if ctx.StatusReporter != nil {
		ctx.StatusReporter.UpdateStatus(status, msg)
	}
}

// bigQuerySource defines the configuration for a single BigQuery query.
type bigQuerySource struct {
	ProjectID          string
	Query              string
	CursorField        string
	CursorInitialValue string
	TimestampField     string
	IdFields           []string
	ExpandJson         bool
}

func (s *bigQuerySource) Name() string {
	// this string uniquely identifies the source in the state store.
	// configuration that doesn't affect the query/cursor itself should not be included.
	name := fmt.Sprintf("%s-%s-%s", s.ProjectID, s.Query, s.CursorField)
	// hash it to avoid unintentionally leaching queries into logs/files
	return fmt.Sprintf("%x", xxhash.Sum64String(name))
}

type bigQueryInput struct {
	config config
	logger *logp.Logger
}

func (bq *bigQueryInput) Name() string {
	return inputName
}

func (bq *bigQueryInput) Test(src cursor.Source, _ v2.TestContext) error {
	return nil
}

func (bq *bigQueryInput) Run(ctx v2.Context, src cursor.Source, cur cursor.Cursor, publisher cursor.Publisher) error {
	updateStatus(ctx, status.Starting, "")

	cancelCtx := v2.GoContextFromCanceler(ctx.Cancelation)

	var opts []option.ClientOption
	if bq.config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(bq.config.CredentialsFile))
	}

	source, _ := src.(*bigQuerySource)
	client, err := bigquery.NewClient(cancelCtx, source.ProjectID, opts...)
	if err != nil {
		err := fmt.Errorf("failed to create bigquery client: %w", err)
		updateStatus(ctx, status.Failed, err.Error())
		return err
	}
	defer client.Close()

	updateStatus(ctx, status.Running, "")
	ticker := time.NewTicker(bq.config.Period)
	defer ticker.Stop()

	for {
		err := bq.querySource(cancelCtx, source, cur, publisher, client)
		if err != nil {
			updateStatus(ctx, status.Degraded, err.Error())
			bq.logger.Error(err.Error())
		}

		select {
		case <-ctx.Cancelation.Done():
			updateStatus(ctx, status.Stopping, "")
			return nil
		case <-ticker.C:
			continue
		}
	}
}

func (bq *bigQueryInput) querySource(ctx context.Context, src *bigQuerySource, cur cursor.Cursor, publisher cursor.Publisher, client *bigquery.Client) error {
	params := make(map[string]interface{})

	if src.CursorField != "" {
		var cursorVal interface{}

		if cur.IsNew() {
			if src.CursorInitialValue != "" {
				cursorVal = src.CursorInitialValue

				// we support expressions in initial cursor values e.g. "TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)".
				// we can't pass this directly as a parameter to BigQuery, so we have to evaluate it first.
				// we ignore errors here; either the value is a literal, or if the expression is invalid the query will fail later anyway.
				query := fmt.Sprintf("SELECT %s AS cursor", src.CursorInitialValue)
				_ = runQuery(ctx, bq.logger, client, query, nil, func(_ bigquery.Schema, row []bigquery.Value) {
					cursorVal = row[0]
				})
			}
		} else {
			lastCursorValue := &cursorState{}
			if err := cur.Unpack(&lastCursorValue); err != nil {
				return fmt.Errorf("failed to unpack cursor: %w", err)
			}

			if val, err := lastCursorValue.get(); err != nil {
				return fmt.Errorf("failed to get cursor value: %w", err)
			} else {
				cursorVal = val
			}
		}

		params["cursor"] = cursorVal
	}

	err := runQuery(ctx, bq.logger, client, src.Query, params, func(schema bigquery.Schema, row []bigquery.Value) {
		bq.publishEvent(src, publisher, schema, row)
	})
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (bq *bigQueryInput) publishEvent(src *bigQuerySource, publisher cursor.Publisher, schema bigquery.Schema, row []bigquery.Value) {
	fields := make(map[string]interface{}, len(row))
	var timestamp time.Time
	var state *cursorState
	idVals := make(map[string]interface{})

	for i, v := range row {
		if v == nil {
			continue
		}

		field := schema[i]

		if src.CursorField != "" && field.Name == src.CursorField {
			cursorState := &cursorState{}
			if err := cursorState.set(field, v); err != nil {
				bq.logger.Error(fmt.Errorf("failed to set cursor state from field '%s': %w", field.Name, err))

			} else {
				bq.logger.Debugf("setting cursor state from field %s", field.Name)
				state = cursorState
			}
		}

		if src.TimestampField != "" && field.Name == src.TimestampField {
			ts, err := getTimestamp(field, v)
			if err == nil {
				bq.logger.Debugf("setting timestamp from field %s", field.Name)
				timestamp = ts
			} else {
				bq.logger.Error(fmt.Errorf("failed to get timestamp from field '%s': %w", field.Name, err))
			}
		}

		if src.ExpandJson {
			val, ok, err := expandJSON(field, v)
			if err == nil {
				if ok {
					bq.logger.Debugf("expanding JSON from field %s", field.Name)
					v = val
				}
			} else {
				// on error, still expand into a nested object with the original string to avoid mapping conflicts
				v = map[string]interface{}{"original": v}
				bq.logger.Error(fmt.Errorf("failed to expand JSON field %s: %w", field.Name, err))
			}
		}

		if slices.Contains(src.IdFields, field.Name) {
			idVals[field.Name] = v
		}

		fields[field.Name] = v
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			// nest everything for now to avoid mapping conflicts in standalone mode
			"bigquery": fields,
		},
	}

	if len(src.IdFields) != 0 {
		// only set the event ID if we have all the specified fields
		if len(src.IdFields) == len(idVals) {
			id := generateEventID(idVals)
			bq.logger.Debugf("setting event ID '%s' from fields %v", id, idVals)
			event.SetID(id)
		} else {
			bq.logger.Warnf("id_fields is configured (%v), but the required fields are not present "+
				"in the query result; falling back to auto-generated ID", src.IdFields)
		}
	}

	// the only error case is if the publisher is closed, which means everything is shutting down anyway
	_ = publisher.Publish(event, state)
}

// generateEventID creates a deterministic ID from the specified fields.
func generateEventID(fields map[string]interface{}) string {
	sorted := slices.Sorted(maps.Keys(fields))

	parts := make([]string, len(fields))
	for i, k := range sorted {
		parts[i] = fmt.Sprintf("%s:%v", k, fields[k])
	}

	return fmt.Sprintf("%x", xxhash.Sum64String(strings.Join(parts, "|")))
}
