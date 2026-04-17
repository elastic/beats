// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package salesforce implements the Filebeat "salesforce" input, which
// collects events from the Salesforce Event Monitoring API.
//
// # Overview
//
// The input authenticates against a Salesforce org using OAuth2 (either the
// JWT Bearer flow or the User-Password flow) and then runs one or both of the
// supported collection methods at a configurable interval:
//
//   - EventLogFile: queries the EventLogFile SObject for newly available log
//     files and downloads each referenced CSV, emitting one event per CSV row.
//   - Object: issues a SOQL query against any Real-Time Event Monitoring
//     object (LoginEvent, LogoutEvent, ApiEvent, ...) and emits one event per
//     returned row.
//
// Both methods are driven by user-supplied SOQL templates (see the QueryConfig
// type and the module-provided default/value templates). The "default"
// template is used the first time the input runs; the "value" template is
// used on every subsequent run and receives the persisted cursor via the
// [[ .cursor ]] variable. The user also configures a cursor field (for
// example, EventDate or LogDate) that the input reads from each returned
// record to advance its watermark.
//
// # Collection lifecycle
//
// Run is the input entry point required by the v2 input framework. On first
// invocation for a given source it starts with an empty state; on subsequent
// invocations the state is restored from the cursor input manager. After
// setting up the Salesforce session and SOQL client, run performs an
// immediate initial collection for each enabled method and then enters a
// ticker-driven loop. Each method has its own independent ticker and its own
// per-method backoff: a failed collection suppresses the next tick for the
// configured interval so a persistent error (for example, a revoked token)
// cannot saturate the Salesforce API.
//
// # Cursor state
//
// State is persisted through the Filebeat cursor input manager as a
// [state] struct containing one [dateTimeCursor] per method
// (object + event_log_file):
//
//   - first_event_time - timestamp of the first event seen during the most
//     recent successful query for the method. This is the field the legacy
//     module templates (LoginEvent / LogoutEvent) use as their resume point,
//     because those SObjects only allow ORDER BY EventDate DESC so the first
//     row of the response is the newest and its timestamp must drive the
//     next query.
//   - last_event_time - timestamp of the last event seen during the most
//     recent successful query. Retained so templates that can sort ascending
//     (EventLogFile, ApiEvent, ...) can resume from there.
//   - progress_time - bounded-batch watermark, written only by the batched
//     object collection path. Records how far into the backlog the input has
//     advanced, independent of individual event timestamps.
//
// The cursor is also exposed to user templates, so a template can branch on
// which fields are set (typically: use progress_time if present, else fall
// back to first/last_event_time, else use an [[ now ]] expression for the
// very first run).
//
// # Bounded batching
//
// Unbatched object collection issues a single open-ended SOQL query every
// interval and relies on Salesforce returning records in cursor-field order.
// That works for real-time modules where the backlog is always a few seconds
// behind, but it does not bound catch-up when the input has been offline for
// hours or when an org produces a large burst of events: a single query can
// return far more rows than the input can publish before the next tick.
//
// When event_monitoring_method.object.batch.enabled is true the input
// switches to a bounded, windowed catch-up strategy. Each ticker runs up to
// batch.max_windows_per_run queries, each covering a (Start, End] slice of
// width batch.window advancing from progress_time (or, on the very first
// run, runEnd - batch.initial_interval) toward the current run time. The
// batched cursor map exposes two additional template variables per window,
// batch_start_time and batch_end_time, which the module-provided value
// template uses to bound the SOQL query's time range explicitly. See
// [objectBatchWindow] and (*salesforceInput).nextObjectBatchWindow in
// batch.go for the window-selection logic.
//
// # Upgrade safety
//
// The batched path was introduced after the input was already shipping in
// the login / logout real-time modules, so existing installs carry legacy
// dateTimeCursor state with first_event_time / last_event_time populated but
// progress_time empty. nextObjectBatchWindow seeds the first post-upgrade
// window from whichever of progress_time, first_event_time or
// last_event_time is set, in that order, before falling back to
// runEnd - initial_interval. Without that fallback the first batched run on
// an upgraded install would either skip events (legacy watermark is older
// than runEnd - initial_interval) or replay them (legacy watermark is
// newer), depending on how long ago the input last ran.
//
// runObjectBatches additionally snapshots the object dateTimeCursor before
// each batched query and restores it on error, so a failed paginated window
// retries the exact same (Start, End] bounds instead of re-advancing from
// first/last_event_time values partially updated mid-window by
// runObjectQuery.
//
// The reverse transition is also migration-safe: if a user disables
// batching after having persisted progress_time, objectCursor projects
// first_event_time / last_event_time as the later of the legacy watermark
// and progress_time. This prevents the unbatched query from replaying quiet
// windows that batching already drained, while still letting subsequent
// unbatched runs move beyond the old batched watermark naturally.
//
// # Fault tolerance
//
// Three defenses protect the input against common failure modes:
//
//   - ctxTransport (input.go) wraps the http.Client transport so every
//     outgoing Salesforce request inherits the input's cancellation
//     context. This matters because go-sfdc builds SOQL requests with
//     http.NewRequest (no context), which would otherwise be unaffected by
//     Filebeat / Elastic Agent shutdown. With the wrap, SIGTERM aborts an
//     in-flight SOQL query and the input returns promptly instead of
//     waiting for the transport-level timeout.
//   - isAuthError + (*salesforceInput).reopenSession provide single-retry
//     recovery from a mid-run authentication failure on the initial SOQL
//     query request and on EventLogFile CSV downloads. The Salesforce access
//     token is obtained once at Setup and is never refreshed by go-sfdc; if
//     it expires or is revoked while the input is running, the input
//     detects INVALID_SESSION_ID / INVALID_AUTH_HEADER / raw 401 signals,
//     opens a fresh session, and retries once. Paginated QueryResult.Next()
//     calls are a known limitation: if the token expires after the first
//     page, the current tick fails and the next scheduled run resumes from
//     the last durable cursor.
//   - formatCollectionStatus surfaces per-method consecutive-failure counts
//     in the Elastic Agent status line (status.Degraded) once the second
//     consecutive failure occurs, and logs a recovery line when a run
//     succeeds after a failure streak. Combined with the existing
//     fixed one-interval cool-off in run() (skipping work until the next
//     interval after a failure), this makes sustained outages visible to
//     operators without drowning the logs in boilerplate.
//
// Cursor durability: the input publishes events via input-cursor's
// Publisher, which only persists cursor state to the registry after the
// associated event is ACKed by the output. A crash or hard-stop between
// publish and ACK is therefore safe: on restart the input resumes from the
// last ACKed cursor, duplicating any in-flight events rather than losing
// them (at-least-once semantics).
//
// # EventLogFile specifics
//
// EventLogFile collection is a two-step process: the SOQL query returns
// rows that each point at a downloadable CSV through a LogFile URL, and the
// input then fetches and parses each referenced CSV. Rows are published in
// a streaming fashion to avoid buffering very large log files in memory;
// processCSVRecords is the shared streaming helper used by both the
// publish path and the live-test decoder. The EventLogFile cursor is
// advanced only after the whole CSV has been consumed so that a mid-file
// parse/publish failure forces the entire file to be retried on the next
// tick.
//
// # File layout
//
//   - input.go         - salesforceInput, Run/Setup, RunObject, RunEventLogFile.
//   - batch.go         - object bounded-batch window selection and cursor parsing.
//   - config.go        - config structs, defaults, and Validate.
//   - config_auth.go   - OAuth2 (JWT bearer + user-password) config.
//   - state.go         - persisted cursor shape and cursor template parsing.
//   - input_manager.go - v2 InputManager and cursor wiring.
//   - soql.go          - SOQL querier implementing go-sfdc's QueryFormatter.
//   - value_tpl.go     - template engine shared by "default" / "value" queries.
//   - helper.go        - small utilities (timeNow indirection, generic pointer helper).
package salesforce
