// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"

	"github.com/osquery/osquery-go/plugin/logger"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type SnapshotResult struct {
	Action       string              `json:"action"`
	Name         string              `json:"name"`
	Numeric      string              `json:"numeric"`
	CalendarTime string              `json:"calendarTime"`
	UnixTime     int64               `json:"unixTime"`
	Hits         []map[string]string `json:"snapshot"`
}

type osqueryLogMessage struct {
	Severity     int    `json:"s"`
	Filename     string `json:"f"`
	Line         int    `json:"i"`
	Message      string `json:"m"`
	CalendarTime string `json:"c"`
	UnixTime     uint64 `json:"u"`
}

const osqueryLogMessageFieldsCount = 6

type osqLogSeverity int

// The severity levels are taken from osquery source
// https://github.com/osquery/osquery/blob/master/osquery/core/plugins/logger.h#L39
//  enum StatusLogSeverity {
// 	  O_INFO = 0,
// 	  O_WARNING = 1,
// 	  O_ERROR = 2,
// 	  O_FATAL = 3,
//  };
const (
	severityInfo osqLogSeverity = iota
	severityWarning
	severityError
	severityFatal
)

func (m *osqueryLogMessage) Log(typ logger.LogType, log *logp.Logger) {
	if log == nil {
		return
	}
	args := make([]interface{}, 0, osqueryLogMessageFieldsCount*2)
	args = append(args, "osquery.log_type")
	args = append(args, typ)
	args = append(args, "osquery.severity")
	args = append(args, m.Severity)
	args = append(args, "osquery.filename")
	args = append(args, m.Filename)
	args = append(args, "osquery.line")
	args = append(args, m.Line)
	args = append(args, "osquery.cal_time")
	args = append(args, m.CalendarTime)
	args = append(args, "osquery.time")
	args = append(args, m.UnixTime)

	switch osqLogSeverity(m.Severity) {
	case severityError, severityFatal:
		log.Errorw(m.Message, args...)
	case severityWarning:
		log.Warnw(m.Message, args...)
	case severityInfo:
		log.Infow(m.Message, args...)
	default:
		log.Debugw(m.Message, args...)
	}
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
		if typ == logger.LogTypeStatus {
			var m osqueryLogMessage
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
