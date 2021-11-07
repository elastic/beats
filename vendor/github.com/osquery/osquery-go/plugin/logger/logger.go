// Package logger creates an osquery logging plugin.
//
// See https://osquery.readthedocs.io/en/latest/development/logger-plugins/ for more.
package logger

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/osquery/osquery-go/gen/osquery"
)

// LogFunc is the logger function used by an osquery Logger plugin.
//
// The LogFunc should log the provided result string. The LogType
// argument can be optionally used to log differently depending on the
// type of log received. The context argument can optionally be used
// for cancellation in long-running operations.
type LogFunc func(ctx context.Context, typ LogType, log string) error

// Plugin is an osquery logger plugin.
// The Plugin struct implements the OsqueryPlugin interface.
type Plugin struct {
	name  string
	logFn LogFunc
}

// NewPlugin takes a value that implements LoggerPlugin and wraps it with
// the appropriate methods to satisfy the OsqueryPlugin interface. Use this to
// easily create plugins implementing osquery loggers.
func NewPlugin(name string, fn LogFunc) *Plugin {
	return &Plugin{name: name, logFn: fn}
}

func (t *Plugin) Name() string {
	return t.name
}

func (t *Plugin) RegistryName() string {
	return "logger"
}

func (t *Plugin) Routes() osquery.ExtensionPluginResponse {
	return []map[string]string{}
}

func (t *Plugin) Ping() osquery.ExtensionStatus {
	return osquery.ExtensionStatus{Code: 0, Message: "OK"}
}

func (t *Plugin) Call(ctx context.Context, request osquery.ExtensionPluginRequest) osquery.ExtensionResponse {
	var err error
	if log, ok := request["string"]; ok {
		err = t.logFn(ctx, LogTypeString, log)
	} else if log, ok := request["snapshot"]; ok {
		err = t.logFn(ctx, LogTypeSnapshot, log)
	} else if log, ok := request["health"]; ok {
		err = t.logFn(ctx, LogTypeHealth, log)
	} else if log, ok := request["init"]; ok {
		err = t.logFn(ctx, LogTypeInit, log)
	} else if _, ok := request["status"]; ok {
		statusJSON := []byte(request["log"])
		if len(statusJSON) == 0 {
			return osquery.ExtensionResponse{
				Status: &osquery.ExtensionStatus{
					Code:    1,
					Message: "got empty status",
				},
			}
		}

		// Dirty hack because osquery gives us malformed JSON.
		statusJSON = bytes.Replace(statusJSON, []byte(`"":`), []byte(``), -1)
		statusJSON[0] = '['
		statusJSON[len(statusJSON)-1] = ']'

		var parsedStatuses []json.RawMessage
		if err := json.Unmarshal(statusJSON, &parsedStatuses); err != nil {
			return osquery.ExtensionResponse{
				Status: &osquery.ExtensionStatus{
					Code:    1,
					Message: "error parsing status logs: " + err.Error(),
				},
			}
		}

		for _, s := range parsedStatuses {
			err = t.logFn(ctx, LogTypeStatus, string(s))
		}
	} else {
		return osquery.ExtensionResponse{
			Status: &osquery.ExtensionStatus{
				Code:    1,
				Message: "unknown log request",
			},
		}
	}

	if err != nil {
		return osquery.ExtensionResponse{
			Status: &osquery.ExtensionStatus{
				Code:    1,
				Message: "error logging: " + err.Error(),
			},
		}
	}

	return osquery.ExtensionResponse{
		Status:   &osquery.ExtensionStatus{Code: 0, Message: "OK"},
		Response: osquery.ExtensionPluginResponse{},
	}
}

func (t *Plugin) Shutdown() {}

//LogType encodes the type of log osquery is outputting.
type LogType int

const (
	LogTypeString LogType = iota
	LogTypeSnapshot
	LogTypeHealth
	LogTypeInit
	LogTypeStatus
)

// String implements the fmt.Stringer interface for LogType.
func (l LogType) String() string {
	var typeString string
	switch l {
	case LogTypeString:
		typeString = "string"
	case LogTypeSnapshot:
		typeString = "snapshot"
	case LogTypeHealth:
		typeString = "health"
	case LogTypeInit:
		typeString = "init"
	case LogTypeStatus:
		typeString = "status"
	default:
		typeString = "unknown"
	}
	return typeString
}
