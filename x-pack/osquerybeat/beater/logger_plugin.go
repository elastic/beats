// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"

	"github.com/osquery/osquery-go/plugin/logger"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqlog"
	"github.com/elastic/elastic-agent-libs/logp"
)

type QueryResult struct {
	Action       string              `json:"action"`
	Name         string              `json:"name"`
	Numeric      bool                `json:"numeric"`
	CalendarTime string              `json:"calendarTime"`
	UnixTime     int64               `json:"unixTime"`
	Epoch        int64               `json:"epoch"`
	Counter      int64               `json:"counter"`
	Hits         []map[string]string `json:"snapshot"`
	DiffResults  struct {
		Added   []map[string]string `json:"added"`
		Removed []map[string]string `json:"removed"`
	} `json:"diffResults"`
}

type HandleQueryResultFunc func(res QueryResult)

type LoggerPlugin struct {
	log           *logp.Logger
	logSnapshotFn HandleQueryResultFunc
}

func NewLoggerPlugin(log *logp.Logger, logSnapshotFn HandleQueryResultFunc) *LoggerPlugin {
	return &LoggerPlugin{
		log:           log.With("ctx", "logger"),
		logSnapshotFn: logSnapshotFn,
	}
}

func (p *LoggerPlugin) Log(ctx context.Context, typ logger.LogType, logText string) error {
	if typ == logger.LogTypeSnapshot || typ == logger.LogTypeString {
		var res QueryResult
		if err := json.Unmarshal([]byte(logText), &res); err != nil {
			p.log.Errorf("failed to unmarshal shapshot result: %v", err)
			return err
		}

		if p.logSnapshotFn != nil {
			p.logSnapshotFn(res)
		}
	} else {
		if typ == logger.LogTypeStatus {
			var m osqlog.LogMessage
			if err := json.Unmarshal([]byte(logText), &m); err != nil {
				p.log.Errorf("failed to unmarshal osquery log message: %v", err)
				return err
			}
			m.Log(typ, p.log)
		} else {
			p.log.Debug(logText)
		}
	}

	return nil
}
