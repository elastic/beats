// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package osqlog centralizes osquery logging helpers used across osquerybeat.
//
// It provides:
// - Severity and Level mappings between osquery/glog and Elastic logging
// - LogMessage for osquery logger plugin JSON payloads (status logs)
// - GlogEntry and ParseGlogLine to parse osqueryd stdout/stderr glog lines
// - LogWithSeverity and LogWithLevel helpers to write structured logs consistently
package osqlog

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"

	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// osquery uses Google glog format: I0314 15:24:36.123456 12345 file.cpp:123] message
	// Level (I/W/E) + MMDD + HH:MM:SS.microseconds + thread_id + file:line] message
	glogPattern = regexp.MustCompile(`^([IWE])(\d{4})\s+(\d{2}:\d{2}:\d{2}\.\d{6})\s+(\d+)\s+([^:]+):(\d+)\]\s*(.*)$`)
)

// Severity represents osquery log severity levels
// The severity levels are taken from osquery source:
// https://github.com/osquery/osquery/blob/master/osquery/core/plugins/logger.h#L39
//
//	enum StatusLogSeverity {
//	    O_INFO = 0,
//	    O_WARNING = 1,
//	    O_ERROR = 2,
//	    O_FATAL = 3,
//	};
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityFatal
)

// LogMessage represents an osquery status log message delivered by the osquery
// logger plugin as JSON. Field tags match osquery's compact JSON keys.
//
// Typical usage:
//
//	var m osqlog.LogMessage
//	_ = json.Unmarshal(raw, &m)
//	m.Log(logger.LogTypeStatus, log)
//
// The Log method writes to the provided logger with consistent structured fields
// under the "osquery.*" namespace and maps osquery severities to Elastic levels.
type LogMessage struct {
	Severity     int    `json:"s"`
	Filename     string `json:"f"`
	Line         int    `json:"i"`
	Message      string `json:"m"`
	CalendarTime string `json:"c"`
	UnixTime     uint64 `json:"u"`
}

const logMessageFieldsCount = 6

// Log writes the LogMessage to the provided logger with mapped severity and
// structured fields. The provided log type is added as a field.
func (m *LogMessage) Log(typ logger.LogType, log *logp.Logger) {
	if log == nil {
		return
	}
	args := make([]any, 0, logMessageFieldsCount*2)
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

	LogWithSeverity(log, Severity(m.Severity), m.Message, args...)
}

// Level represents glog-style log level (single character: I, W, E).
// It can be converted to Severity via ToSeverity.
type Level string

const (
	LevelInfo    Level = "I"
	LevelWarning Level = "W"
	LevelError   Level = "E"
)

// ToSeverity converts a glog Level to the corresponding Severity.
func (l Level) ToSeverity() Severity {
	switch l {
	case LevelError:
		return SeverityError
	case LevelWarning:
		return SeverityWarning
	case LevelInfo:
		return SeverityInfo
	default:
		return SeverityInfo
	}
}

// LogWithSeverity logs a message at the appropriate Elastic Agent log level
// that corresponds to the given osquery Severity. Additional fields may be
// provided as key/value pairs in args.
func LogWithSeverity(log *logp.Logger, severity Severity, message string, args ...any) {
	if log == nil {
		return
	}
	switch severity {
	case SeverityError, SeverityFatal:
		log.Errorw(message, args...)
	case SeverityWarning:
		log.Warnw(message, args...)
	default:
		log.Debugw(message, args...)
	}
}

// LogWithLevel logs at a level mapped from the provided glog Level.
func LogWithLevel(log *logp.Logger, level Level, message string, args ...any) {
	LogWithSeverity(log, level.ToSeverity(), message, args...)
}

// GlogEntry represents a parsed osquery glog entry emitted on osqueryd
// stdout/stderr. It captures the level, timestamp, source, thread id and message.
type GlogEntry struct {
	level      Level     // I, W, E
	timestamp  time.Time // parsed timestamp
	threadID   int       // thread ID
	sourceFile string    // source file name
	sourceLine int       // source line number
	message    string    // log message
}

// Log writes the GlogEntry to the provided logger including structured fields
// under the "osquery.*" namespace with a level mapped from GlogEntry.Level.
func (e *GlogEntry) Log(log *logp.Logger) {
	if log == nil {
		return
	}
	args := []any{
		"osquery.timestamp", e.timestamp,
		"osquery.thread_id", e.threadID,
		"osquery.source.file", e.sourceFile,
		"osquery.source.line", e.sourceLine,
	}
	LogWithLevel(log, e.level, e.message, args...)
}

// ParseGlogLine parses an osquery log line in glog format from osqueryd
// stdout/stderr.
//
// Format: I0314 15:24:36.123456 12345 file.cpp:123] message
//
// Timestamps are constructed in UTC, inferring the year dynamically by picking
// the closest of now.Year()-1, now.Year(), or now.Year()+1 to avoid issues
// around the new year boundary.
func ParseGlogLine(line string) (*GlogEntry, error) {
	return ParseGlogLineWithNow(line, time.Now().UTC())
}

// ParseGlogLineWithNow is like ParseGlogLine but allows injecting the reference
// time "now" for deterministic tests.
func ParseGlogLineWithNow(line string, now time.Time) (*GlogEntry, error) {
	matches := glogPattern.FindStringSubmatch(line)
	if matches == nil || len(matches) != 8 {
		return nil, fmt.Errorf("line does not match osquery log format")
	}

	entry := &GlogEntry{
		level:      Level(matches[1]),
		sourceFile: matches[5],
		message:    matches[7],
	}

	// Parse thread ID
	threadID, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse thread ID: %w", err)
	}
	entry.threadID = threadID

	// Parse source line
	sourceLine, err := strconv.Atoi(matches[6])
	if err != nil {
		return nil, fmt.Errorf("failed to parse source line: %w", err)
	}
	entry.sourceLine = sourceLine

	// Parse timestamp (month/day and time-of-day)
	monthDay := matches[2]
	if len(monthDay) != 4 {
		return nil, fmt.Errorf("invalid month-day format")
	}
	month, err := strconv.Atoi(monthDay[:2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse month: %w", err)
	}
	day, err := strconv.Atoi(monthDay[2:4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse day: %w", err)
	}

	timeStr := matches[3]
	tod, err := time.Parse("15:04:05.999999", timeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %w", err)
	}

	// Infer year based on current month to handle year boundaries:
	// - If we're in January and see December, assume previous year
	// - If we're in December and see January, assume next year
	// - Otherwise, use current year
	year := now.Year()
	logMonth := time.Month(month)
	nowMonth := now.Month()

	if nowMonth == time.January && logMonth == time.December {
		year--
	} else if nowMonth == time.December && logMonth == time.January {
		year++
	}

	entry.timestamp = time.Date(
		year,
		logMonth,
		day,
		tod.Hour(),
		tod.Minute(),
		tod.Second(),
		tod.Nanosecond(),
		time.UTC,
	)

	return entry, nil
}
