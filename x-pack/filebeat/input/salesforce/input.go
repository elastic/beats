// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/elastic/go-sfdc"
	"github.com/elastic/go-sfdc/credentials"
	"github.com/elastic/go-sfdc/session"
	"github.com/elastic/go-sfdc/soql"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/go-concert/ctxtool"
)

const (
	inputName         = "salesforce"
	formatRFC3339Like = "2006-01-02T15:04:05.999Z"
)

// salesforceInput is the runtime state of a single configured input
// instance. The embedded config is the unpacked user configuration; srcConfig
// is a pointer to the same value, retained separately so run-time helpers can
// nil-check the pointer without re-validating the embedded struct. All
// timestamps, including the persisted cursor reachable through cursor, are
// kept in UTC.
type salesforceInput struct {
	ctx           context.Context
	publisher     inputcursor.Publisher
	cancel        context.CancelCauseFunc
	cursor        *state
	srcConfig     *config
	sfdcConfig    *sfdc.Configuration
	log           *logp.Logger
	clientSession *session.Session
	soqlr         *soql.Resource
	config
}

// // The Filebeat user-agent is provided to the program as useragent.
// var userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

// Plugin returns the input plugin.
func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:      inputName,
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}
}

func (s *salesforceInput) Name() string { return inputName }

func (s *salesforceInput) Test(_ inputcursor.Source, _ v2.TestContext) error {
	return nil
}

// Run starts the input and blocks until it ends completes. It will return on
// context cancellation or type invalidity errors, any other error will be retried.
func (s *salesforceInput) Run(env v2.Context, src inputcursor.Source, cursor inputcursor.Cursor, pub inputcursor.Publisher) (err error) {

	env.UpdateStatus(status.Starting, "Initializing Salesforce input")
	st := &state{}
	if !cursor.IsNew() {
		if err = cursor.Unpack(&st); err != nil {
			env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to set up Salesforce input: %v", err))
			return err
		}
	}

	env.UpdateStatus(status.Configuring, "Configuring Salesforce input")
	if err = s.Setup(env, src, st, pub); err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to set up Salesforce input: %v", err))
		return err
	}
	env.UpdateStatus(status.Running, "Salesforce input setup complete. Monitoring events.")
	return s.run(env)
}

// Setup sets up the input. It will create a new SOQL resource and all other
// necessary configurations.
func (s *salesforceInput) Setup(env v2.Context, src inputcursor.Source, cursor *state, pub inputcursor.Publisher) (err error) {
	srcSource, ok := src.(*source)
	if !ok {
		return fmt.Errorf("failed to assert src as *source")
	}

	cfg := srcSource.cfg

	ctx := ctxtool.FromCanceller(env.Cancelation)
	childCtx, cancel := context.WithCancelCause(ctx)

	s.srcConfig = &cfg
	s.ctx = childCtx
	s.cancel = cancel
	s.publisher = pub
	s.cursor = cursor
	s.log = env.Logger.With("input_url", cfg.URL)
	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	if err != nil {
		return fmt.Errorf("error with configuration: %w", err)
	}

	s.soqlr, err = s.SetupSFClientConnection() // create a new SOQL resource
	if err != nil {
		return fmt.Errorf("error setting up connection to Salesforce: %w", err)
	}

	return nil
}

// run is the main loop of the input. It will run until the context is cancelled
// and based on the configuration, it will run the different methods -- EventLogFile
// or Object to collect events at defined intervals.
func (s *salesforceInput) run(env v2.Context) error {
	s.log.Info("Starting Salesforce input run")
	defer func() {
		env.UpdateStatus(status.Stopped, "Salesforce input stopped")
	}()
	if s.srcConfig == nil || s.srcConfig.EventMonitoringMethod == nil {
		return errors.New("internal error: salesforce monitoring configuration is not set")
	}
	ctx := s.ctx
	if ctx == nil {
		return errors.New("internal error: salesforce context is not set")
	}
	// elfFails / objectFails track the number of consecutive failures for
	// each collection method across collection iterations within this
	// run() invocation. The counter is included in the Degraded status
	// message (and the error log) once it exceeds 1 so an operator watching
	// agent status can distinguish a transient hiccup from a sustained
	// outage without having to cross-reference log timestamps. Counters
	// reset to zero on the first successful run after a failure streak.
	var elfFails, objectFails int
	// eventLogFileBackoffUntil / objectBackoffUntil are declared up here so a
	// failure during the immediate startup-phase collection (below) can
	// install the cooldown that the ticker loop reads. Without that, the
	// first ticker tick after a startup failure always processed a fresh
	// request, doubling Salesforce API pressure during a sustained outage.
	var eventLogFileBackoffUntil, objectBackoffUntil time.Time

	if s.srcConfig.EventMonitoringMethod.EventLogFile.isEnabled() {
		err := s.RunEventLogFile()
		if err != nil {
			elfFails++
			env.UpdateStatus(status.Degraded, formatCollectionStatus("EventLogFile", elfFails, err))
			s.log.Errorf("Problem running EventLogFile collection (consecutive failures: %d): %s", elfFails, err)
			eventLogFileBackoffUntil = nextBackoffUntil(s.srcConfig.EventMonitoringMethod.EventLogFile.Interval)
		} else {
			elfFails = 0
			s.log.Info("Initial EventLogFile collection completed successfully")
		}
	}

	if s.srcConfig.EventMonitoringMethod.Object.isEnabled() {
		err := s.RunObject()
		if err != nil {
			objectFails++
			env.UpdateStatus(status.Degraded, formatCollectionStatus("Object", objectFails, err))
			s.log.Errorf("Problem running Object collection (consecutive failures: %d): %s", objectFails, err)
			objectBackoffUntil = nextBackoffUntil(s.srcConfig.EventMonitoringMethod.Object.Interval)
		} else {
			objectFails = 0
			s.log.Info("Initial Object collection completed successfully")
		}
	}

	eventLogFileTicker, objectMethodTicker := &time.Ticker{}, &time.Ticker{}
	eventLogFileTicker.C, objectMethodTicker.C = nil, nil

	if s.srcConfig.EventMonitoringMethod.EventLogFile.isEnabled() {
		eventLogFileTicker = time.NewTicker(s.srcConfig.EventMonitoringMethod.EventLogFile.Interval)
		defer eventLogFileTicker.Stop()
	}

	if s.srcConfig.EventMonitoringMethod.Object.isEnabled() {
		objectMethodTicker = time.NewTicker(s.srcConfig.EventMonitoringMethod.Object.Interval)
		defer objectMethodTicker.Stop()
	}

	for {
		// Always check for cancel first, to not accidentally trigger another
		// run if the context is already cancelled, but we have already received
		// another ticker making the channel ready.
		select {
		case <-ctx.Done():
			env.UpdateStatus(status.Stopping, "Salesforce input stopping")
			return s.isError(ctx.Err())
		default:
		}

		select {
		case <-ctx.Done():
			env.UpdateStatus(status.Stopping, "Salesforce input stopping")
			return s.isError(ctx.Err())
		case <-eventLogFileTicker.C:
			if !eventLogFileBackoffUntil.IsZero() && time.Now().Before(eventLogFileBackoffUntil) {
				s.log.Debugf("Skipping EventLogFile collection until %s after previous failure", eventLogFileBackoffUntil.Format(time.RFC3339Nano))
				continue
			}
			s.log.Info("Running EventLogFile collection")
			if err := s.RunEventLogFile(); err != nil {
				elfFails++
				env.UpdateStatus(status.Degraded, formatCollectionStatus("EventLogFile", elfFails, err))
				s.log.Errorf("Problem running EventLogFile collection (consecutive failures: %d): %s", elfFails, err)
				eventLogFileBackoffUntil = nextBackoffUntil(s.srcConfig.EventMonitoringMethod.EventLogFile.Interval)
			} else {
				if elfFails > 0 {
					s.log.Infof("EventLogFile collection recovered after %d consecutive failures", elfFails)
				}
				elfFails = 0
				eventLogFileBackoffUntil = time.Time{}
				env.UpdateStatus(status.Running, "EventLogFile collection completed successfully")
				s.log.Info("EventLogFile collection completed successfully")
			}
		case <-objectMethodTicker.C:
			if !objectBackoffUntil.IsZero() && time.Now().Before(objectBackoffUntil) {
				s.log.Debugf("Skipping Object collection until %s after previous failure", objectBackoffUntil.Format(time.RFC3339Nano))
				continue
			}
			s.log.Info("Running Object collection")
			if err := s.RunObject(); err != nil {
				objectFails++
				env.UpdateStatus(status.Degraded, formatCollectionStatus("Object", objectFails, err))
				s.log.Errorf("Problem running Object collection (consecutive failures: %d): %s", objectFails, err)
				objectBackoffUntil = nextBackoffUntil(s.srcConfig.EventMonitoringMethod.Object.Interval)
			} else {
				if objectFails > 0 {
					s.log.Infof("Object collection recovered after %d consecutive failures", objectFails)
				}
				objectFails = 0
				objectBackoffUntil = time.Time{}
				env.UpdateStatus(status.Running, "Object collection completed successfully")
				s.log.Info("Object collection completed successfully")
			}
		}
	}
}

// formatCollectionStatus renders the Degraded status message surfaced to
// Elastic Agent when a collection run fails. The consecutive-failure count
// is included once fails > 1 so a single transient failure still reads
// naturally while a sustained outage becomes visually distinct.
func formatCollectionStatus(method string, fails int, err error) string {
	if fails > 1 {
		return fmt.Sprintf("Error running %s collection (%d consecutive failures): %v", method, fails, err)
	}
	return fmt.Sprintf("Error running %s collection: %v", method, err)
}

// nextBackoffUntil returns the wall-clock time the next ticker tick should be
// suppressed until after a failed collection. The naive choice "now + interval"
// races the ticker: time.NewTicker(interval) keeps firing on the original
// cadence, so the next tick arrives at almost exactly the same instant the
// backoff would otherwise expire and the Before-check evaluates to false. We
// pad the window by half an interval so the next tick is reliably suppressed
// while the tick after that is still let through, delivering the
// "one-tick-suppressed" cadence promised by doc.go (i.e. one failed Salesforce
// API call per 2 * interval during a sustained outage instead of one per
// interval).
func nextBackoffUntil(interval time.Duration) time.Time {
	return time.Now().Add(interval + interval/2)
}

func (s *salesforceInput) isError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		s.log.Infof("input stopped because context was cancelled with: %v", err)
		return nil
	}

	return err
}

// isAuthError reports whether err looks like a Salesforce authentication
// failure, i.e. a 401 Unauthorized response or a canonical Salesforce auth
// error code. It is intentionally string-based because go-sfdc flattens the
// underlying *http.Response into a free-form error message (see
// soql.Resource.queryResponse) and does not export a typed sentinel.
//
// Matches (in priority order):
//   - INVALID_SESSION_ID: the canonical Salesforce error code returned when
//     the access token is expired, revoked, or otherwise invalid.
//   - INVALID_AUTH_HEADER: returned when the Authorization header is
//     malformed, which covers the "no active session" case go-sfdc produces
//     before an initial token fetch succeeds.
//   - ": 401 " / "status code 401": fallback for raw-status responses
//     without a Salesforce-shaped JSON error body.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "INVALID_SESSION_ID") ||
		strings.Contains(msg, "INVALID_AUTH_HEADER") ||
		strings.Contains(msg, ": 401 ") ||
		strings.Contains(msg, "status code 401")
}

// reopenSession acquires a fresh Salesforce session and SOQL resource,
// replacing s.clientSession and s.soqlr on success. It is called by the
// SOQL / ELF-download paths when they see an auth error so a token that
// was revoked or expired mid-run can be replaced without restarting the
// input.
//
// reopenSession is not safe for concurrent use; the input's Run loop is
// single-goroutine today and callers are expected to invoke it serially
// during a failed query's error-handling path.
func (s *salesforceInput) reopenSession() error {
	if s.sfdcConfig == nil {
		return errors.New("internal error: salesforce configuration is not set")
	}
	newSess, err := session.Open(*s.sfdcConfig)
	if err != nil {
		return fmt.Errorf("failed to re-open salesforce session: %w", err)
	}
	newSoqlr, err := soql.NewResource(newSess)
	if err != nil {
		return fmt.Errorf("failed to re-create SOQL resource on new session: %w", err)
	}
	s.clientSession = newSess
	s.soqlr = newSoqlr
	s.log.Info("Salesforce session re-opened after auth failure")
	return nil
}

// SetupSFClientConnection opens an authenticated Salesforce session using
// the previously prepared sfdcConfig, stores it on the receiver for reuse
// (EventLogFile CSV downloads reuse the session for the Authorization
// header), and returns a SOQL resource bound to it.
func (s *salesforceInput) SetupSFClientConnection() (*soql.Resource, error) {
	s.log.Info("Setting up Salesforce client connection")
	if s.sfdcConfig == nil {
		return nil, errors.New("internal error: salesforce configuration is not set properly")
	}

	// Open creates a session using the configuration.
	session, err := session.Open(*s.sfdcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open salesforce connection: %w", err)
	}
	s.log.Info("Salesforce session opened successfully")

	// Set clientSession for re-use.
	s.clientSession = session

	// Create a new SOQL resource using the session.
	soqlr, err := soql.NewResource(session)
	if err != nil {
		return nil, fmt.Errorf("error setting up salesforce SOQL resource: %w", err)
	}
	return soqlr, nil
}

// FormQueryWithCursor takes a queryConfig and a cursor and returns a querier.
func (s *salesforceInput) FormQueryWithCursor(queryConfig *QueryConfig, cursor mapstr.M) (*querier, error) {
	qr, err := parseCursor(queryConfig, cursor, s.log)
	if err != nil {
		return nil, err
	}

	return &querier{Query: qr}, err
}

// isZero checks if the given value v is the zero value for its type.
// It compares v to the zero value obtained by new(T).
func isZero[T comparable](v T) bool {
	return v == *new(T)
}

// objectConfig returns the Object method's config block with nil guards on
// the receiver, srcConfig, and EventMonitoringMethod. A non-nil error here
// is a programming bug (Setup should have rejected the configuration) and
// is surfaced verbatim so callers can include it in their own error chains.
func (s *salesforceInput) objectConfig() (*EventMonitoringConfig, error) {
	if s == nil || s.srcConfig == nil || s.srcConfig.EventMonitoringMethod == nil {
		return nil, errors.New("internal error: object monitoring configuration is not set")
	}
	return &s.srcConfig.EventMonitoringMethod.Object, nil
}

// eventLogFileConfig is the EventLogFile counterpart to objectConfig.
func (s *salesforceInput) eventLogFileConfig() (*EventMonitoringConfig, error) {
	if s == nil || s.srcConfig == nil || s.srcConfig.EventMonitoringMethod == nil {
		return nil, errors.New("internal error: event log file monitoring configuration is not set")
	}
	return &s.srcConfig.EventMonitoringMethod.EventLogFile, nil
}

// RunObject runs the Object method of the Event Monitoring API to collect events.
func (s *salesforceInput) RunObject() error {
	objectCfg, err := s.objectConfig()
	if err != nil {
		return err
	}
	if s.cursor == nil {
		// Defensive initialization for tests/callers that invoke RunObject
		// directly without going through the full setup path.
		s.cursor = &state{}
	}

	s.log.Infof("Running Object collection with interval: %s", objectCfg.Interval)

	if objectCfg.Batch.isEnabled() {
		return s.runObjectBatches()
	}

	// Snapshot the in-memory object cursor and revert on error, mirroring
	// runObjectBatches. runObjectQuery mutates first_event_time /
	// last_event_time / last_event_id per row before publish, so a transient
	// publish/query failure mid-stream would otherwise leave the cursor
	// advanced past rows that were never durably ACKed and the next tick
	// would skip them.
	prevCursor := s.cursor.Object
	totalEvents, err := s.runObjectQuery(s.objectCursor(nil))
	if err != nil {
		s.cursor.Object = prevCursor
		return err
	}
	s.log.Infof("Total events: %d", totalEvents)

	return nil
}

// objectCursor builds the template context map exposed to the Object
// query's "value" template. It returns nil when no cursor has been
// persisted yet and batch is nil, so the caller falls back to the
// "default" template on the very first run.
//
// When batch is non-nil (bounded batching), batch_start_time and
// batch_end_time are added under the "object" key. first_event_time,
// last_event_time and progress_time are forwarded whenever they are
// populated, so a template can prefer progress_time on resume but still
// fall back to the legacy watermarks after an upgrade.
//
// When batch is nil (unbatched collection) and progress_time exists from a
// previous batched run, first_event_time / last_event_time are projected as
// the later of the legacy watermark and progress_time. This keeps a user who
// disables batching from replaying quiet time ranges that batching has
// already advanced through, while still allowing later unbatched runs to move
// beyond the old progress_time using their native legacy cursor fields.
// last_event_id is also exposed on the unbatched path, even when empty, so
// templates can use it as an optional tie-breaker for non-unique timestamp
// cursor fields without breaking old persisted state that predates it.
func (s *salesforceInput) objectCursor(batch *objectBatchWindow) mapstr.M {
	var cursor mapstr.M
	if !isZero(s.cursor.Object.FirstEventTime) || !isZero(s.cursor.Object.LastEventTime) || !isZero(s.cursor.Object.ProgressTime) || batch != nil {
		object := make(mapstr.M)
		firstEventTime := s.cursor.Object.FirstEventTime
		lastEventTime := s.cursor.Object.LastEventTime
		if batch == nil && !isZero(s.cursor.Object.ProgressTime) {
			firstEventTime = laterObjectResumeWatermark(firstEventTime, s.cursor.Object.ProgressTime)
			lastEventTime = laterObjectResumeWatermark(lastEventTime, s.cursor.Object.ProgressTime)
		}
		if !isZero(firstEventTime) {
			object.Put("first_event_time", firstEventTime)
		}
		if !isZero(lastEventTime) {
			object.Put("last_event_time", lastEventTime)
		}
		if batch == nil || !isZero(s.cursor.Object.LastEventID) {
			object.Put("last_event_id", s.cursor.Object.LastEventID)
		}
		// Batched object collection advances with progress_time. first/last_event_time
		// still reflect the observed events from the most recent successful window
		// so existing templates keep working, but they are not the batching cursor.
		if !isZero(s.cursor.Object.ProgressTime) {
			object.Put("progress_time", s.cursor.Object.ProgressTime)
		}
		if batch != nil {
			object.Put("batch_start_time", formatBatchCursorTime(batch.Start))
			object.Put("batch_end_time", formatBatchCursorTime(batch.End))
		}
		cursor = mapstr.M{"object": object}
	}

	return cursor
}

// laterObjectResumeWatermark returns whichever of legacyWatermark or
// progressTime represents the later point in time. It is used when a user
// disables batching after previously persisting progress_time so unbatched
// templates that still reference first_event_time / last_event_time resume
// from the latest safe watermark rather than replaying already-drained batch
// windows.
//
// This comparison is best-effort: if either value cannot be parsed, the
// legacy watermark is returned unchanged to avoid introducing a new failure
// path for pre-existing state that older releases would have passed through
// verbatim to the template.
func laterObjectResumeWatermark(legacyWatermark, progressTime string) string {
	if isZero(progressTime) {
		return legacyWatermark
	}
	progressTS, err := parseBatchCursorTime(progressTime)
	if err != nil {
		return legacyWatermark
	}
	if isZero(legacyWatermark) {
		return progressTime
	}
	legacyTS, err := parseBatchCursorTime(legacyWatermark)
	if err != nil {
		return legacyWatermark
	}
	if progressTS.After(legacyTS) {
		return progressTime
	}
	return legacyWatermark
}

// runObjectBatches drives bounded-batch Object collection. It issues up to
// batch.max_windows_per_run bounded SOQL queries per tick, advancing
// progress_time at the end of each successful (Start, End] window.
//
// Each window is computed from the latest persisted cursor by
// nextObjectBatchWindow, which also applies the upgrade-safety fallback
// from legacy first_event_time / last_event_time watermarks. The object
// cursor is snapshotted before each query and restored on error, so a
// failed paginated window retries the exact same bounds on the next tick
// instead of re-advancing from first/last_event_time values partially
// updated mid-window by runObjectQuery.
//
// The loop also stops early once a window reaches runEnd, because any
// further window would be empty.
func (s *salesforceInput) runObjectBatches() error {
	objectCfg, err := s.objectConfig()
	if err != nil {
		return err
	}

	runEnd := timeNow().UTC()
	totalEvents := 0

	for i := 0; i < objectCfg.Batch.getMaxWindowsPerRun(); i++ {
		window, ok, err := s.nextObjectBatchWindow(runEnd)
		if err != nil {
			return fmt.Errorf("error building object batch window: %w", err)
		}
		if !ok {
			break
		}

		prevCursor := s.cursor.Object
		count, err := s.runObjectQuery(s.objectCursor(&window))
		if err != nil {
			s.cursor.Object = prevCursor
			return err
		}
		totalEvents += count
		s.cursor.Object.ProgressTime = formatBatchCursorTime(window.End)

		if !window.End.Before(runEnd) {
			break
		}
	}

	s.log.Infof("Total events: %d", totalEvents)

	return nil
}

// runObjectQuery renders the Object query from the provided template
// context, issues it against Salesforce, walks every page of results, and
// publishes one event per record. It returns the total number of events
// published.
//
// Per-row side effects:
//
//   - first_event_time is written from the cursor field of the first row of
//     the first page only (legacy semantics required for ORDER BY EventDate
//     DESC real-time objects, where the first row is the newest).
//   - last_event_time is written from every row, overwriting on each call,
//     so at end-of-query it reflects the final row seen.
//   - last_event_id is reset to empty at the start of the run and then
//     written from the record Id whenever a row exposes one. Rows without an
//     Id do not clear a previously observed one within the same run. This
//     lets same-timestamp ascending queries resume from the last ACKed row
//     instead of skipping later rows that share the same timestamp, while
//     never carrying a stale Id across runs that no longer return one.
//
// Any mid-stream error aborts immediately and returns the count published
// so far. The caller is responsible for deciding whether to keep or revert
// those partial cursor mutations (runObjectBatches reverts on error to keep
// retry semantics stable).
func (s *salesforceInput) runObjectQuery(cursor mapstr.M) (int, error) {
	objectCfg, err := s.objectConfig()
	if err != nil {
		return 0, err
	}
	if objectCfg.Query == nil || objectCfg.Cursor == nil || objectCfg.Cursor.Field == "" {
		return 0, errors.New("internal error: object query/cursor configuration is not set")
	}

	query, err := s.FormQueryWithCursor(objectCfg.Query, cursor)
	if err != nil {
		return 0, fmt.Errorf("error forming query based on cursor: %w", err)
	}

	s.log.Infof("Query formed: %s", query.Query)

	res, err := s.queryWithReauth(query)
	if err != nil {
		return 0, err
	}

	totalEvents := 0
	firstEvent := true
	// Reset LastEventID at the start of each run so stale values from a
	// previous query (e.g. after a user changes their SOQL to omit Id) can
	// never be carried forward as a tie-breaker into a new last_event_time
	// bucket. Rows with an Id below will repopulate it.
	s.cursor.Object.LastEventID = ""

	for res.TotalSize() > 0 {
		for _, rec := range res.Records() {
			val := rec.Record().Fields()

			jsonStrEvent, err := json.Marshal(val)
			if err != nil {
				return 0, err
			}

			if timestamp, ok := val[objectCfg.Cursor.Field].(string); ok {
				if firstEvent {
					s.cursor.Object.FirstEventTime = timestamp
				}
				s.cursor.Object.LastEventTime = timestamp
			}
			if id, ok := val["Id"].(string); ok {
				s.cursor.Object.LastEventID = id
			}

			err = publishEvent(s.publisher, s.cursor, jsonStrEvent, "Object")
			if err != nil {
				return 0, err
			}
			firstEvent = false
			totalEvents++
		}

		if !res.MoreRecords() { // returns true if there are more records.
			break
		}

		res, err = res.Next()
		if err != nil {
			return 0, err
		}
	}

	return totalEvents, nil
}

// RunEventLogFile runs the EventLogFile method of the Event Monitoring API to
// collect events.
func (s *salesforceInput) RunEventLogFile() error {
	eventLogFileCfg, err := s.eventLogFileConfig()
	if err != nil {
		return err
	}
	if s.cursor == nil {
		// Defensive initialization for tests/callers that invoke RunEventLogFile
		// directly without going through the full setup path.
		s.cursor = &state{}
	}
	if eventLogFileCfg.Query == nil || eventLogFileCfg.Cursor == nil || eventLogFileCfg.Cursor.Field == "" {
		return errors.New("internal error: event log file query/cursor configuration is not set")
	}

	s.log.Infof("Running EventLogFile collection with interval: %s", eventLogFileCfg.Interval)

	var cursor mapstr.M
	if !isZero(s.cursor.EventLogFile.FirstEventTime) || !isZero(s.cursor.EventLogFile.LastEventTime) {
		eventLogFile := make(mapstr.M)
		if !isZero(s.cursor.EventLogFile.FirstEventTime) {
			eventLogFile.Put("first_event_time", s.cursor.EventLogFile.FirstEventTime)
		}
		if !isZero(s.cursor.EventLogFile.LastEventTime) {
			eventLogFile.Put("last_event_time", s.cursor.EventLogFile.LastEventTime)
		}
		eventLogFile.Put("last_event_id", s.cursor.EventLogFile.LastEventID)
		cursor = mapstr.M{"event_log_file": eventLogFile}
	}

	query, err := s.FormQueryWithCursor(eventLogFileCfg.Query, cursor)
	if err != nil {
		return fmt.Errorf("error forming query based on cursor: %w", err)
	}

	s.log.Infof("Query formed: %s", query.Query)

	res, err := s.queryWithReauth(query)
	if err != nil {
		return err
	}

	// NOTE: This is a failsafe check because the HTTP client is always set.
	// This check allows unit tests to verify correct behavior when the HTTP
	// client is nil.
	if s.sfdcConfig == nil || s.sfdcConfig.Client == nil {
		return errors.New("internal error: salesforce configuration is not set properly")
	}

	totalEvents, firstEvent := 0, true
	for res.TotalSize() > 0 {
		for _, rec := range res.Records() {
			logfile, ok := rec.Record().Fields()["LogFile"].(string)
			if !ok {
				return fmt.Errorf("LogFile field not found or not a string in Salesforce event log file: %v", rec.Record().Fields())
			}

			published, err := s.fetchAndPublishLogFile(logfile)
			if err != nil {
				return err
			}

			// Advance the EventLogFile cursor only after the whole CSV stream is
			// processed successfully. This avoids skipping unread rows when a
			// mid-stream parse/publish failure happens.
			if timestamp, ok := rec.Record().Fields()[eventLogFileCfg.Cursor.Field].(string); ok {
				if firstEvent {
					s.cursor.EventLogFile.FirstEventTime = timestamp
				}
				s.cursor.EventLogFile.LastEventTime = timestamp
				s.cursor.EventLogFile.LastEventID = ""
				if id, ok := rec.Record().Fields()["Id"].(string); ok {
					s.cursor.EventLogFile.LastEventID = id
				}
			}

			totalEvents += published
			firstEvent = false
		}

		if !res.MoreRecords() {
			break
		}

		res, err = res.Next()
		if err != nil {
			return fmt.Errorf("error getting next page: %w", err)
		}
	}
	s.log.Infof("Total events processed: %d", totalEvents)

	return nil
}

// queryWithReauth issues the given SOQL query and, if the response looks
// like an auth failure (see isAuthError), re-opens the Salesforce session
// and retries exactly once. Pagination follow-ups (QueryResult.Next) are
// NOT retried here because go-sfdc's result object retains a pointer to
// the old session; if a 401 surfaces mid-pagination it is allowed to
// bubble up and the next run() tick retries the window from scratch,
// which then exercises this helper from the top.
func (s *salesforceInput) queryWithReauth(query *querier) (*soql.QueryResult, error) {
	res, err := s.soqlr.Query(query, false)
	if err == nil {
		return res, nil
	}
	if !isAuthError(err) {
		return nil, err
	}
	s.log.Warnw("SOQL query failed with auth error; re-opening session and retrying once",
		"error", err)
	if reopenErr := s.reopenSession(); reopenErr != nil {
		return nil, fmt.Errorf("auth error (%w) and session re-open failed: %w", err, reopenErr)
	}
	return s.soqlr.Query(query, false)
}

// fetchAndPublishLogFile downloads the referenced EventLogFile CSV and
// streams each row into the publisher via publishCSVRecords. If the server
// responds with 401 on the first attempt, reopenSession is called and the
// download is retried exactly once; all other non-200 statuses are surfaced
// to the caller. The logfile path is taken verbatim from the LogFile field
// on the EventLogFile record and is appended to s.URL as the go-sfdc
// library does not expose a helper for this.
func (s *salesforceInput) fetchAndPublishLogFile(logfile string) (int, error) {
	resp, err := s.downloadLogFileOnce(logfile)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		s.log.Warn("EventLogFile download returned 401; re-opening session and retrying once")
		if reopenErr := s.reopenSession(); reopenErr != nil {
			return 0, fmt.Errorf("log file download got 401 and session re-open failed: %w", reopenErr)
		}
		resp, err = s.downloadLogFileOnce(logfile)
		if err != nil {
			return 0, err
		}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return 0, fmt.Errorf("unexpected status code %d for log file", resp.StatusCode)
	}
	published, err := s.publishCSVRecords(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, fmt.Errorf("error processing log file CSV: %w", err)
	}
	return published, nil
}

// downloadLogFileOnce issues a single GET against the EventLogFile
// download URL, attaching the current session's authorization header. The
// caller is responsible for closing resp.Body and for handling any retry
// semantics (401, 5xx, etc.). It is split from fetchAndPublishLogFile so
// the retry loop can reuse the same request construction without risk of
// consuming the body twice.
func (s *salesforceInput) downloadLogFileOnce(logfile string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, s.URL+logfile, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for log file: %w", err)
	}
	s.clientSession.AuthorizationHeader(req)
	// NOTE: If we ever see a production issue related to this, then only
	// we should consider adding the header: "X-PrettyPrint:1". See:
	// https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/dome_event_log_file_download.htm?q=X-PrettyPrint%3A1
	resp, err := s.sfdcConfig.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching log file: %w", err)
	}
	return resp, nil
}

// normalizeOAuthTokenURL accepts either a Salesforce OAuth host
// ("https://login.salesforce.com" or "https://your-domain.my.salesforce.com")
// or the canonical token endpoint without query parameters or fragments
// ("https://login.salesforce.com/services/oauth2/token"), and returns the base
// URL shape expected by go-sfdc before it appends "/services/oauth2/token".
func normalizeOAuthTokenURL(rawURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	trimmed = strings.TrimSuffix(trimmed, "/services/oauth2/token")
	return strings.TrimRight(trimmed, "/")
}

// getSFDCConfig returns a new Salesforce configuration based on the configuration.
func (s *salesforceInput) getSFDCConfig(cfg *config) (*sfdc.Configuration, error) {
	var (
		creds *credentials.Credentials
		err   error
	)

	if cfg.Auth == nil {
		return nil, errors.New("no auth provider enabled")
	}
	if cfg.Auth.OAuth2 == nil {
		return nil, errors.New("no auth provider enabled")
	}

	switch {
	case cfg.Auth.OAuth2.JWTBearerFlow != nil && cfg.Auth.OAuth2.JWTBearerFlow.isEnabled():
		s.log.Info("Using JWT Bearer Flow for authentication")
		pemBytes, err := os.ReadFile(cfg.Auth.OAuth2.JWTBearerFlow.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("problem with client key path for JWT auth: %w", err)
		}

		signKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("problem with client key for JWT auth: %w", err)
		}

		passCreds := credentials.JwtCredentials{
			URL:            cfg.Auth.OAuth2.JWTBearerFlow.URL,
			TokenURL:       normalizeOAuthTokenURL(cfg.Auth.OAuth2.JWTBearerFlow.TokenURL),
			ClientId:       cfg.Auth.OAuth2.JWTBearerFlow.ClientID,
			ClientUsername: cfg.Auth.OAuth2.JWTBearerFlow.ClientUsername,
			ClientKey:      signKey,
		}

		creds, err = credentials.NewJWTCredentials(passCreds)
		if err != nil {
			return nil, fmt.Errorf("error creating jwt credentials: %w", err)
		}

	case cfg.Auth.OAuth2.UserPasswordFlow != nil && cfg.Auth.OAuth2.UserPasswordFlow.isEnabled():
		s.log.Info("Using User Password Flow for authentication")
		passCreds := credentials.PasswordCredentials{
			URL:          normalizeOAuthTokenURL(cfg.Auth.OAuth2.UserPasswordFlow.TokenURL),
			Username:     cfg.Auth.OAuth2.UserPasswordFlow.Username,
			Password:     cfg.Auth.OAuth2.UserPasswordFlow.Password,
			ClientID:     cfg.Auth.OAuth2.UserPasswordFlow.ClientID,
			ClientSecret: cfg.Auth.OAuth2.UserPasswordFlow.ClientSecret,
		}

		creds, err = credentials.NewPasswordCredentials(passCreds)
		if err != nil {
			return nil, fmt.Errorf("error creating password credentials: %w", err)
		}

	}

	client, err := newClient(*cfg, s.inputCtx, s.log)
	if err != nil {
		return nil, fmt.Errorf("problem with client: %w", err)
	}

	return &sfdc.Configuration{
		Credentials: creds,
		Client:      client,
		Version:     cfg.Version,
	}, nil
}

// inputCtx returns the input's live cancellation context, or
// context.Background when Setup has not run yet. Used by the ctxTransport
// wrapper so that every outgoing Salesforce HTTP request (including SOQL
// queries that go-sfdc builds with http.NewRequest rather than
// NewRequestWithContext) inherits input cancellation.
func (s *salesforceInput) inputCtx() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// ctxTransport is an http.RoundTripper wrapper that clones every outgoing
// request with a context supplied by getCtx. It lets the salesforce input
// propagate input-level cancellation into HTTP calls built by third-party
// code (notably go-sfdc's SOQL queries, which use http.NewRequest and do
// not thread any context through). When getCtx returns nil the request is
// forwarded unchanged so tests that bypass Setup are unaffected.
type ctxTransport struct {
	rt     http.RoundTripper
	getCtx func() context.Context
}

// RoundTrip implements http.RoundTripper.
func (t *ctxTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.getCtx != nil {
		if ctx := t.getCtx(); ctx != nil {
			req = req.Clone(ctx)
		}
	}
	return t.rt.RoundTrip(req)
}

// retryLog is a shim for the retryablehttp.Client.Logger.
type retryLog struct{ log *logp.Logger }

func newRetryLog(log *logp.Logger) *retryLog {
	return &retryLog{log: log.Named("retryablehttp").WithOptions(zap.AddCallerSkip(1))}
}

func (l *retryLog) Error(msg string, kv ...interface{}) { l.log.Errorw(msg, kv...) }
func (l *retryLog) Info(msg string, kv ...interface{})  { l.log.Infow(msg, kv...) }
func (l *retryLog) Debug(msg string, kv ...interface{}) { l.log.Debugw(msg, kv...) }
func (l *retryLog) Warn(msg string, kv ...interface{})  { l.log.Warnw(msg, kv...) }

// retryErrorHandler returns a retryablehttp.ErrorHandler that will log retry resignation
// but return the last retry attempt's response and a nil error to allow the retryablehttp.Client
// evaluate the response status itself. Any error passed to the retryablehttp.ErrorHandler
// is returned unaltered. Despite not being documented so, the error handler may be passed
// a nil resp. retryErrorHandler will handle this case.
func retryErrorHandler(max int, log *logp.Logger) retryablehttp.ErrorHandler {
	return func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		if resp != nil && resp.Request != nil {
			reqURL := "unavailable"
			if resp.Request.URL != nil {
				reqURL = resp.Request.URL.String()
			}
			log.Warnw("giving up retries", "method", resp.Request.Method, "url", reqURL, "retries", max+1)
		} else {
			log.Warnw("giving up retries: no response available", "retries", max+1)
		}
		return resp, err
	}
}

// newClient builds the Salesforce HTTP client. getCtx, when non-nil, is
// used to inject the input's live cancellation context into every outgoing
// request so graceful shutdown aborts in-flight SOQL queries immediately
// instead of waiting for transport.Timeout.
func newClient(cfg config, getCtx func() context.Context, log *logp.Logger) (*http.Client, error) {
	c, err := cfg.Resource.Transport.Client(httpcommon.WithLogger(log))
	if err != nil {
		return nil, err
	}

	// Wrap the inner transport so ctx cancellation reaches requests built
	// by go-sfdc without passing through http.NewRequestWithContext. This
	// wraps BEFORE retryablehttp so the injection happens on every retry
	// attempt, not just the first.
	if getCtx != nil {
		if c.Transport == nil {
			c.Transport = http.DefaultTransport
		}
		c.Transport = &ctxTransport{rt: c.Transport, getCtx: getCtx}
	}

	if maxAttempts := cfg.Resource.Retry.getMaxAttempts(); maxAttempts > 1 {
		c = (&retryablehttp.Client{
			HTTPClient:   c,
			Logger:       newRetryLog(log),
			RetryWaitMin: cfg.Resource.Retry.getWaitMin(),
			RetryWaitMax: cfg.Resource.Retry.getWaitMax(),
			RetryMax:     maxAttempts,
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
			ErrorHandler: retryErrorHandler(maxAttempts, log),
		}).StandardClient()

		// BUG: retryablehttp ignores the timeout previously set. So, setting it
		// again.
		c.Timeout = cfg.Resource.Transport.Timeout

		// Wrap the StandardClient transport so the input's cancellation
		// context is attached to the request before retryablehttp's
		// CheckRetry / Backoff inspect req.Context().Err(). The inner
		// ctxTransport set above on the original http.Transport is not
		// enough on its own: retryablehttp wraps the http.Client, and the
		// requests it sees are built by go-sfdc with http.NewRequest (no
		// context). Without this outer wrap, retryablehttp treats every
		// cancelled-by-inner-ctxTransport attempt as a retryable network
		// error and runs the full backoff before giving up, so SIGTERM
		// during an in-flight SOQL query waits ~30s instead of returning
		// promptly.
		if getCtx != nil {
			c.Transport = &ctxTransport{rt: c.Transport, getCtx: getCtx}
		}
	}

	return c, nil
}

// publishEvent publishes an event using the configured publisher pub.
func publishEvent(pub inputcursor.Publisher, cursor *state, jsonStrEvent []byte, dataCollectionMethod string) error {
	if pub == nil {
		return errors.New("publisher is not set")
	}

	event := beat.Event{
		Timestamp: timeNow(),
		Fields: mapstr.M{
			"message": string(jsonStrEvent),
			"event": mapstr.M{
				"provider": dataCollectionMethod,
			},
		},
	}

	return pub.Publish(event, cursor)
}

type textContextError struct {
	error
	body []byte
}

// processCSVRecords streams a Salesforce EventLogFile CSV from r, reusing
// the underlying csv.Reader row buffer and invoking onRecord once per data
// row with a freshly allocated header-keyed map. Processing stops at the
// first error returned by onRecord (bubbled up unchanged) or at a CSV
// decode error (wrapped with the 1-based row number, counting the header).
// An empty body - or a body with only a header - is not an error and
// returns (0, nil). The function returns the number of successfully
// handled rows, which is the meaningful total even when the caller later
// receives an error, because rows processed before the failure have
// already been emitted.
func (s *salesforceInput) processCSVRecords(r io.Reader, onRecord func(map[string]string) error) (int, error) {
	csvReader := csv.NewReader(r)

	// To share the backing array for performance.
	csvReader.ReuseRecord = true

	// Lazy quotes are enabled to allow for quoted fields with commas. More flexible
	// in handling CSVs.
	// NOTE(shmsr): Although, we didn't face any issue with LazyQuotes == false, but I
	// think we should keep it enabled to avoid any issues in the future.
	csvReader.LazyQuotes = true

	header, err := csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	header = slices.Clone(header)

	count := 0
	rowNum := 1
	for {
		rowNum++
		record, err := csvReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return count, nil
			}
			return count, fmt.Errorf("failed to read CSV row %d: %w for: %v", rowNum, err, record)
		}

		event := make(map[string]string, len(header))
		for i, h := range header {
			event[h] = record[i]
		}

		if err := onRecord(event); err != nil {
			return count, err
		}
		count++
	}
}

// publishCSVRecords streams an EventLogFile CSV body from r and publishes
// each row as an event tagged event.provider="EventLogFile". It is the
// primary production helper for EventLogFile parsing; the full-buffer
// decodeAsCSV variant exists only for tests that need to inspect the
// parsed rows directly.
//
// Streaming means earlier rows can reach the publisher before a later row
// causes an error. That tradeoff is accepted (and covered by tests) to
// avoid buffering arbitrarily large EventLogFile bodies in memory. The
// EventLogFile cursor is advanced by the caller only after the whole file
// has been processed, so a partial failure forces the entire file to be
// re-fetched and re-published on the next tick.
func (s *salesforceInput) publishCSVRecords(r io.Reader) (int, error) {
	return s.processCSVRecords(r, func(val map[string]string) error {
		jsonStrEvent, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("error json marshaling event: %w", err)
		}

		if err := publishEvent(s.publisher, s.cursor, jsonStrEvent, "EventLogFile"); err != nil {
			return fmt.Errorf("error publishing event: %w", err)
		}
		return nil
	})
}

// decodeAsCSV decodes the provided byte slice as a CSV and returns a slice of
// maps, where each map represents a row in the CSV with the header fields as
// keys and the row values as values.
func (s *salesforceInput) decodeAsCSV(p []byte) ([]map[string]string, error) {
	var results []map[string]string //nolint:prealloc // not sure about the size to prealloc with
	_, err := s.processCSVRecords(bytes.NewReader(p), func(event map[string]string) error {
		results = append(results, event)
		return nil
	})
	if err != nil {
		s.log.Errorf("failed to decode CSV: %v\n%s", err, p)
		return nil, textContextError{error: err, body: p}
	}
	return results, nil
}
