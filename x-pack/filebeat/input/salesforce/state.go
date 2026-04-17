// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// state is the state of the salesforce module. It is used to watermark the state
// to avoid pulling duplicate data from Salesforce. The state is persisted separately
// for EventLogFile and Object.
type state struct {
	Object       dateTimeCursor `json:"object,omitempty"`
	EventLogFile dateTimeCursor `json:"event_log_file,omitempty"`
}

// dateTimeCursor maintains the persisted cursor state for one collection method.
// first/last_event_time retain the observed event timestamps from the most recent
// successful query, while progress_time tracks bounded catch-up progress when
// batching is enabled.
// In certain SOQL queries for specific objects, sorting by all fields may not be feasible, and there may be no specific order.
// This design allows users to exert maximum control over the iteration process.
// For instance, the LoginEvent object only supports sorting based on EventIdentifier and EventDate.
// Furthermore, if we desire to sort based on EventDate, it only supports descending order sorting.
// In this case by using first_event_time we can get latest event EventDate to query next set of events.
// Reference to LoginEvent: https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/sforce_api_objects_loginevent.htm
type dateTimeCursor struct {
	// FirstEventTime holds the cursor-field value of the first record seen
	// in the most recent successful query (i.e. the newest, for DESC-only
	// real-time objects like LoginEvent). Legacy real-time modules drive
	// their next query from this field.
	FirstEventTime string `struct:"first_event_time,omitempty"`
	// LastEventTime holds the cursor-field value of the last record seen in
	// the most recent successful query. Used by templates that can sort
	// ascending on the cursor field, such as EventLogFile.
	LastEventTime string `struct:"last_event_time,omitempty"`
	// ProgressTime is the bounded-batch watermark written by the batched
	// Object collection path (see batch.go). It advances in fixed-width
	// windows independently of individual event timestamps and is the
	// preferred resume point when present. It is empty on fresh installs
	// and on installs that only ever used the unbatched real-time path.
	ProgressTime string `struct:"progress_time,omitempty"`
}

// parseCursor parses the cursor from the configuration and executes the
// template. If cursor is nil, the default templated query is used else
// the value templated query is used. See QueryConfig struct for more.
func parseCursor(cfg *QueryConfig, cursor mapstr.M, log *logp.Logger) (string, error) {
	ctxTmpl := mapstr.M{"cursor": nil}

	if cursor != nil {
		ctxTmpl["cursor"] = cursor
		qr, err := cfg.Value.Execute(ctxTmpl, nil, log)
		if err != nil {
			return "", err
		}
		return qr, nil
	}

	return cfg.Default.Execute(ctxTmpl, nil, log)
}
