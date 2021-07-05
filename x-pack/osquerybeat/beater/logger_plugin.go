// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"

	"github.com/kolide/osquery-go/plugin/logger"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type SnapshotResult struct {
	Action       string              `json:"action"`
	Name         string              `json:"name"`
	Numeric      string              `json:"numeric"`
	CalendarTime string              `json:"calendarTime"`
	UnixTime     int64               `json:"unixTime"`
	Hits         []map[string]string `json:"snapshot"`
}

type HandleSnapshotResultFunc func(res SnapshotResult)

type LoggerPlugin struct {
	log           *logp.Logger
	logSnapshotFn HandleSnapshotResultFunc
}

func NewLoggerPlugin(log *logp.Logger, logSnapshotFn HandleSnapshotResultFunc) *LoggerPlugin {
	return &LoggerPlugin{
		log:           log.With("ctx", "logger"),
		logSnapshotFn: logSnapshotFn,
	}
}

func (p *LoggerPlugin) Log(ctx context.Context, typ logger.LogType, logText string) error {
	if typ == logger.LogTypeSnapshot {
		var res SnapshotResult
		if err := json.Unmarshal([]byte(logText), &res); err != nil {
			p.log.Errorf("failed to unmarshal shapshot result: %v", err)
			return err
		}
		if p.logSnapshotFn != nil {
			p.logSnapshotFn(res)
		}
	} else {
		raw := []byte(logText)
		p.log.Debugf("log type: %s, %s", typ, string(raw))
	}

	return nil
}
