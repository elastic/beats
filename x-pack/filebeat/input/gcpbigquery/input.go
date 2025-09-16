package gcpbigquery

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"google.golang.org/api/option"
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
		sources = append(sources, &bigQuerySource{
			ProjectID:   config.ProjectID,
			Query:       query,
			CursorField: config.CursorField,
		})
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
	ProjectID   string
	Query       string
	CursorField string
}

func (s *bigQuerySource) Name() string {
	return fmt.Sprintf("%s-%s", s.ProjectID, s.Query)
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
	bq.logger.Infof("starting BigQuery input") // how to reference the source without logging the query?
	updateStatus(ctx, status.Starting, "")

	cancelCtx := v2.GoContextFromCanceler(ctx.Cancelation)

	var opts []option.ClientOption
	if bq.config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(bq.config.CredentialsFile))
	}

	source := src.(*bigQuerySource)
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
	query := src.Query

	if src.CursorField != "" {
		where := ""
		sort := fmt.Sprintf("ORDER BY %s ASC", src.CursorField)

		if !cur.IsNew() {
			lastCursorValue := &cursorState{}
			if err := cur.Unpack(&lastCursorValue); err != nil {
				return fmt.Errorf("failed to unpack cursor: %w", err)
			}

			if lastCursorValue.WhereVal != "" {
				where = fmt.Sprintf("WHERE %s > %s", src.CursorField, lastCursorValue.WhereVal)
			}
		}

		// this is not efficient but allows us to wrap arbitrary queries.
		// perhaps later we can properly parse and modify the SQL AST
		query = fmt.Sprintf("SELECT * FROM (%s) %s %s", src.Query, where, sort)
	}

	err := runQuery(ctx, bq.logger, client, query, func(schema bigquery.Schema, row []bigquery.Value) {
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

	for i, v := range row {
		if v == nil {
			continue
		}

		field := schema[i]

		if src.CursorField != "" && field.Name == src.CursorField {
			bq.logger.Debugf("setting cursor state from field %s", field.Name)

			cursorState := &cursorState{}
			err := cursorState.set(field, v)
			if err == nil {
				state = cursorState
			} else {
				bq.logger.Error(fmt.Errorf("failed to set cursor state from field '%s': %w", field.Name, err))
			}
		}

		if bq.config.TimestampField != "" && field.Name == bq.config.TimestampField {
			bq.logger.Debugf("setting timestamp from field %s", field.Name)

			ts, err := getTimestamp(field, v)
			if err == nil {
				timestamp = ts
			} else {
				bq.logger.Error(fmt.Errorf("failed to get timestamp from field '%s': %w", field.Name, err))
			}
		}

		if bq.config.ExpandJsonStrings {
			bq.logger.Debugf("expanding JSON from field %s", field.Name)

			val, err := expandJSON(field, v)
			if err == nil {
				v = val
			} else {
				// on error, still expand into a nested object with the original string to avoid mapping conflicts
				v = map[string]interface{}{"original": v}
				bq.logger.Error(fmt.Errorf("failed to expand JSON field %s: %w", field.Name, err))
			}
		}

		fields[field.Name] = v
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	publisher.Publish(beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			// nest everything for now to avoid mapping conflicts in standalone mode
			"bigquery": fields,
		},
	}, state)
}
