// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build salesforce_live
// +build salesforce_live

// Local-only live Salesforce validation.
//
// Credentials: copy x-pack/filebeat/input/salesforce/sf-creds.env.example to
// x-pack/filebeat/input/salesforce/sf-creds.env (gitignored), then set standardized keys:
// SALESFORCE_INSTANCE_URL, SALESFORCE_CLIENT_ID, SALESFORCE_CLIENT_SECRET,
// SALESFORCE_USERNAME, SALESFORCE_PASSWORD, SALESFORCE_SECURITY_TOKEN, SALESFORCE_TOKEN_URL.
//
// EventLogFile rows are backed by hourly log files; Salesforce typically materializes a
// file roughly on the order of an hour after the activity period, not within a short poll
// after API/tooling calls. This suite therefore does not assert immediate ELF rows from
// generated activity in the same run; Login/Logout ELF tests use historical rows (or skip
// when absent), and Apex ELF uses historical EventLogFile metadata plus cursor advancement.
//
// Example:
// SALESFORCE_LIVE_CREDS_FILE=./x-pack/filebeat/input/salesforce/sf-creds.env \
// go test -tags salesforce_live -timeout 30m \
//   -run TestLiveSalesforceValidation -v ./x-pack/filebeat/input/salesforce
//
// With SALESFORCE_LIVE_GENERATE_ACTIVITY=1, use an absolute path to the creds file if
// the relative path is not resolved from your working directory, and keep -timeout
// generous (logout object polling may run up to 5 minutes).

package salesforce

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

type liveSalesforceCreds struct {
	InstanceURL   string
	ClientID      string
	ClientSecret  string
	Username      string
	Password      string
	SecurityToken string
	TokenURL      string
}

func liveSalesforceCredsPath() string {
	if path := os.Getenv("SALESFORCE_LIVE_CREDS_FILE"); path != "" {
		return path
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "sf-creds.env"
	}

	return filepath.Join(filepath.Dir(file), "sf-creds.env")
}

func loadLiveSalesforceCreds(t *testing.T) liveSalesforceCreds {
	t.Helper()

	path := liveSalesforceCredsPath()
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("skipping local Salesforce live validation: credentials file not available at %s", path)
		}
		require.NoError(t, err, "expected Salesforce credentials path to be accessible")
	}

	creds, err := parseLiveSalesforceCredsFile(path)
	require.NoError(t, err, "expected valid Salesforce credentials in %s", path)

	return creds
}

func newLiveSalesforceInput(t *testing.T, cfg config) *salesforceInput {
	t.Helper()

	baseCtx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 5*time.Second {
			deadline = deadline.Add(-5 * time.Second)
		}
		var baseCancel context.CancelFunc
		baseCtx, baseCancel = context.WithDeadline(baseCtx, deadline)
		t.Cleanup(baseCancel)
	}
	ctx, cancel := context.WithCancelCause(baseCtx)
	t.Cleanup(func() { cancel(nil) })

	s := &salesforceInput{
		ctx:       ctx,
		cancel:    cancel,
		cursor:    &state{},
		srcConfig: &cfg,
		config:    cfg,
		log:       logp.NewLogger("salesforce_live_validation"),
	}

	var err error
	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected live Salesforce auth config setup to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected live Salesforce query client setup to succeed")

	return s
}

func liveConfigWithMethod(creds liveSalesforceCreds, method eventMonitoringMethod) config {
	cfg := defaultConfig()
	cfg.URL = creds.InstanceURL
	cfg.Auth = &authConfig{
		OAuth2: &OAuth2{
			UserPasswordFlow: &UserPasswordFlow{
				Enabled:      pointer(true),
				ClientID:     creds.ClientID,
				ClientSecret: creds.ClientSecret,
				TokenURL:     strings.TrimSuffix(creds.TokenURL, "/services/oauth2/token"),
				Username:     creds.Username,
				Password:     creds.Password + creds.SecurityToken,
			},
		},
	}
	cfg.EventMonitoringMethod = &method
	return cfg
}

func eventFieldValues(t *testing.T, events []string, field string) []string {
	t.Helper()

	values := make([]string, 0, len(events))
	for _, raw := range events {
		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &decoded), "expected live event message to be valid JSON")
		value, ok := decoded[field].(string)
		if ok && value != "" {
			values = append(values, value)
		}
	}
	return values
}

func publishedMessages(events []beat.Event) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		message, ok := event.Fields["message"].(string)
		if ok {
			out = append(out, message)
		}
	}
	return out
}

func assertNoDuplicateStrings(t *testing.T, name string, first, second []string) {
	t.Helper()

	seen := make(map[string]struct{}, len(first))
	for _, value := range first {
		seen[value] = struct{}{}
	}

	for _, value := range second {
		_, ok := seen[value]
		assert.False(t, ok, "%s should not emit duplicate values across consecutive live runs", name)
	}
}

func assertCursorNotBefore(t *testing.T, earlier, later string, context string) {
	t.Helper()

	require.NotEmpty(t, earlier, "expected earlier cursor for %s", context)
	require.NotEmpty(t, later, "expected later cursor for %s", context)

	earlierTS, err := parseBatchCursorTime(earlier)
	require.NoError(t, err, "expected earlier cursor time for %s to parse", context)

	laterTS, err := parseBatchCursorTime(later)
	require.NoError(t, err, "expected later cursor time for %s to parse", context)

	assert.False(t, laterTS.Before(earlierTS), "%s cursor should not move backwards", context)
}

func TestLiveSalesforceValidation(t *testing.T) {
	// Generated-activity paths poll LoginEvent for up to liveObjectPollTimeout and
	// LogoutEvent for up to liveLogoutObjectPollTimeout (see live_activity_helpers_test.go).
	// The default go test -timeout is 10m for the whole test, which aborts mid-poll without
	// diagnostics. Require a generous budget when activity generation is enabled.
	if liveActivityGenerationEnabled() {
		if deadline, ok := t.Deadline(); ok && time.Until(deadline) < 17*time.Minute {
			t.Fatalf("SALESFORCE_LIVE_GENERATE_ACTIVITY=1 needs a longer go test -timeout (e.g. -timeout 30m); remaining %v is too small for generated-activity object polling (logout uses the longer poll budget)",
				time.Until(deadline))
		}
	}

	creds := loadLiveSalesforceCreds(t)

	t.Run("login object batching", func(t *testing.T) {
		if !liveActivityGenerationEnabled() {
			t.Skip("set SALESFORCE_LIVE_GENERATE_ACTIVITY=1 to run generated login object batching coverage")
		}

		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: 5 * time.Minute,
				Batch: &batchConfig{
					Enabled:          pointer(true),
					InitialInterval:  24 * time.Hour,
					Window:           12 * time.Hour,
					MaxWindowsPerRun: pointer(2),
				},
				Query: &QueryConfig{
					Default: getValueTpl("SELECT FIELDS(STANDARD) FROM LoginEvent ORDER BY EventDate DESC"),
					Value:   getValueTpl("SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.object.batch_start_time ]] AND EventDate <= [[ .cursor.object.batch_end_time ]] ORDER BY EventDate DESC"),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		})

		// Build the input after the fresh OAuth login so the SOQL session used for
		// polling is not invalidated by generateLoginActivity's password flow.
		session := generateLoginActivity(t, creds)
		s := newLiveSalesforceInput(t, cfg)
		_, err := waitForObjectEvent(t, s, "LoginEvent", session.Window, 0)
		require.NoError(t, err, "expected LoginEvent rows after generated login activity (see poll diagnostics in error)")

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunObject(), "expected live login object batching run to succeed")
		require.NoError(t, requirePositiveGeneratedLiveRows("login object batching first run", len(first.published)))
		firstCursor := s.cursor.Object.ProgressTime
		require.NotEmpty(t, firstCursor, "expected live login object batching to persist progress_time")

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunObject(), "expected live login object batching resume run to succeed")
		secondCursor := s.cursor.Object.ProgressTime
		assertCursorNotBefore(t, firstCursor, secondCursor, "login object batching progress_time")

		firstIDs := eventFieldValues(t, publishedMessages(first.published), "Id")
		secondIDs := eventFieldValues(t, publishedMessages(second.published), "Id")
		assertNoDuplicateStrings(t, "login object batching Ids", firstIDs, secondIDs)
	})

	t.Run("logout object batching", func(t *testing.T) {
		if !liveActivityGenerationEnabled() {
			t.Skip("set SALESFORCE_LIVE_GENERATE_ACTIVITY=1 to run generated logout object batching coverage")
		}

		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: 5 * time.Minute,
				Batch: &batchConfig{
					Enabled:          pointer(true),
					InitialInterval:  24 * time.Hour,
					Window:           12 * time.Hour,
					MaxWindowsPerRun: pointer(2),
				},
				Query: &QueryConfig{
					Default: getValueTpl("SELECT FIELDS(STANDARD) FROM LogoutEvent ORDER BY EventDate DESC"),
					Value:   getValueTpl("SELECT FIELDS(STANDARD) FROM LogoutEvent WHERE EventDate > [[ .cursor.object.batch_start_time ]] AND EventDate <= [[ .cursor.object.batch_end_time ]] ORDER BY EventDate DESC"),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		})

		// Generate logout before creating the input so SOAP logout / best-effort OAuth
		// revoke does not invalidate the REST session used for SOQL polling and RunObject.
		window := generateLogoutActivity(t, creds)
		s := newLiveSalesforceInput(t, cfg)
		_, pollErr := waitForObjectEvent(t, s, "LogoutEvent", window, liveLogoutObjectPollTimeout)
		if pollErr != nil {
			if !errors.Is(pollErr, context.DeadlineExceeded) {
				require.NoError(t, pollErr, "expected logout object polling to succeed; only timeout is treated as an org-capability skip")
			}

			recentCount, newestInOrg, recentQ, maxQ, diagErr := liveProbeLogoutEventRecencyDiagnostics(t, s)
			require.NoError(t, diagErr, "expected logout diagnostics query to succeed after timeout waiting for generated LogoutEvent")
			if recentCount > 0 {
				t.Fatalf("expected generated LogoutEvent to be observed within poll timeout, but the org returned recent LogoutEvent rows [%d] instead of remaining empty: poll_err=%v recent_query=%q newest_EventDate=%q max_query=%q",
					recentCount, pollErr, recentQ, newestInOrg, maxQ)
			}
			t.Skipf("skipping logout object batching: LogoutEvent not observed after generateLogoutActivity (SOAP logout + best-effort OAuth revoke) within poll: %v [org snapshot for context: LogoutEvent rows in last %v=%d recent_query=%q; newest_EventDate=%q max_query=%q]",
				pollErr, liveProbeLogoutEventRecencyWindow, recentCount, recentQ, newestInOrg, maxQ)
		}

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunObject(), "expected live logout object batching run to succeed")
		require.NoError(t, requirePositiveGeneratedLiveRows("logout object batching first run", len(first.published)))
		firstCursor := s.cursor.Object.ProgressTime
		require.NotEmpty(t, firstCursor, "expected live logout object batching to persist progress_time")

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunObject(), "expected live logout object batching resume run to succeed")
		secondCursor := s.cursor.Object.ProgressTime
		assertCursorNotBefore(t, firstCursor, secondCursor, "logout object batching progress_time")

		firstIDs := eventFieldValues(t, publishedMessages(first.published), "Id")
		secondIDs := eventFieldValues(t, publishedMessages(second.published), "Id")
		assertNoDuplicateStrings(t, "logout object batching Ids", firstIDs, secondIDs)
	})

	t.Run("setup audit trail object", func(t *testing.T) {
		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: 5 * time.Minute,
				Query: &QueryConfig{
					Default: getValueTpl(`SELECT FIELDS(STANDARD) FROM SetupAuditTrail WHERE CreatedDate > [[ (formatTime (now.Add (parseDuration "-720h")) "2006-01-02T15:04:05.000Z0700") ]] ORDER BY CreatedDate ASC NULLS FIRST`),
					Value:   getValueTpl(`SELECT FIELDS(STANDARD) FROM SetupAuditTrail WHERE CreatedDate > [[ .cursor.object.last_event_time ]] ORDER BY CreatedDate ASC NULLS FIRST`),
				},
				Cursor: &cursorConfig{Field: "CreatedDate"},
			},
		})

		s := newLiveSalesforceInput(t, cfg)

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunObject(), "expected live setup audit trail object run to succeed")
		assert.Empty(t, s.cursor.Object.ProgressTime, "expected setup audit trail to remain on the non-batched cursor path")
		firstCursor := s.cursor.Object.LastEventTime

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunObject(), "expected live setup audit trail resume run to succeed")
		secondCursor := s.cursor.Object.LastEventTime

		if firstCursor != "" && secondCursor != "" {
			assertCursorNotBefore(t, firstCursor, secondCursor, "setup audit trail last_event_time")
		}

		firstIDs := eventFieldValues(t, publishedMessages(first.published), "Id")
		secondIDs := eventFieldValues(t, publishedMessages(second.published), "Id")
		assertNoDuplicateStrings(t, "setup audit trail Ids", firstIDs, secondIDs)

		if len(first.published) == 0 && len(second.published) == 0 {
			t.Skip("skipping setup audit trail: no live SetupAuditTrail rows were available in the tested window")
		}
	})

	t.Run("login event log file", func(t *testing.T) {
		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			EventLogFile: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Hour,
				Query: &QueryConfig{
					Default: getValueTpl(`SELECT CreatedDate,LogDate,LogFile FROM EventLogFile WHERE CreatedDate > [[ (formatTime (now.Add (parseDuration "-720h")) "2006-01-02T15:04:05.000Z0700") ]] AND EventType = 'Login' ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
					Value:   getValueTpl(`SELECT CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.last_event_time ]] ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
				},
				Cursor: &cursorConfig{Field: "CreatedDate"},
			},
		})

		probeClient := newLiveSalesforceInput(t, cfg)
		loginELFAvailable, loginELFN, loginELFQ, err := liveProbeEventLogFileEventTypeExists(t, probeClient, "Login")
		require.NoError(t, err, "expected EventLogFile Login probe to succeed")
		if !loginELFAvailable {
			t.Skipf("skipping login EventLogFile: no historical EventLogFile rows for EventType='Login' (probe query=%q totalSize=%d). Login ELF is not available on this org.",
				loginELFQ, loginELFN)
		}

		// Historical default query (-720h) + cursor resume; ELF files are hourly and not immediate.
		s := newLiveSalesforceInput(t, cfg)

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunEventLogFile(), "expected live login EventLogFile run to succeed")
		require.NoError(t, requirePositiveHistoricalLiveRows("login EventLogFile first run", len(first.published)))
		firstCursor := s.cursor.EventLogFile.LastEventTime
		require.NotEmpty(t, firstCursor, "expected live login EventLogFile first run to persist last_event_time")

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunEventLogFile(), "expected live login EventLogFile resume run to succeed")
		secondCursor := s.cursor.EventLogFile.LastEventTime

		if firstCursor != "" && secondCursor != "" {
			assertCursorNotBefore(t, firstCursor, secondCursor, "login EventLogFile last_event_time")
		}
	})

	t.Run("logout event log file", func(t *testing.T) {
		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			EventLogFile: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Hour,
				Query: &QueryConfig{
					Default: getValueTpl(`SELECT CreatedDate,LogDate,LogFile FROM EventLogFile WHERE CreatedDate > [[ (formatTime (now.Add (parseDuration "-720h")) "2006-01-02T15:04:05.000Z0700") ]] AND EventType = 'Logout' ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
					Value:   getValueTpl(`SELECT CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Logout' AND CreatedDate > [[ .cursor.event_log_file.last_event_time ]] ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
				},
				Cursor: &cursorConfig{Field: "CreatedDate"},
			},
		})

		probeClient := newLiveSalesforceInput(t, cfg)
		logoutELFAvailable, logoutELFN, logoutELFQ, err := liveProbeEventLogFileEventTypeExists(t, probeClient, "Logout")
		require.NoError(t, err, "expected EventLogFile Logout probe to succeed")
		if !logoutELFAvailable {
			t.Skipf("skipping logout EventLogFile: no historical EventLogFile rows for EventType='Logout' (probe query=%q totalSize=%d). Logout ELF is not available on this org.",
				logoutELFQ, logoutELFN)
		}

		s := newLiveSalesforceInput(t, cfg)

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunEventLogFile(), "expected live logout EventLogFile run to succeed")
		require.NoError(t, requirePositiveHistoricalLiveRows("logout EventLogFile first run", len(first.published)))
		firstCursor := s.cursor.EventLogFile.LastEventTime
		require.NotEmpty(t, firstCursor, "expected live logout EventLogFile first run to persist last_event_time")

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunEventLogFile(), "expected live logout EventLogFile resume run to succeed")
		secondCursor := s.cursor.EventLogFile.LastEventTime

		if firstCursor != "" && secondCursor != "" {
			assertCursorNotBefore(t, firstCursor, secondCursor, "logout EventLogFile last_event_time")
		}
	})

	t.Run("apex event log file", func(t *testing.T) {
		cfg := liveConfigWithMethod(creds, eventMonitoringMethod{
			EventLogFile: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Hour,
				Query: &QueryConfig{
					Default: getValueTpl(`SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE CreatedDate > [[ (formatTime (now.Add (parseDuration "-720h")) "2006-01-02T15:04:05.000Z0700") ]] AND (EventType = 'ApexCallout' OR EventType = 'ApexExecution' OR EventType = 'ApexRestApi' OR EventType = 'ApexSoap' OR EventType = 'ApexTrigger' OR EventType = 'ExternalCustomApexCallout') ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
					Value:   getValueTpl(`SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE CreatedDate > [[ .cursor.event_log_file.last_event_time ]] AND (EventType = 'ApexCallout' OR EventType = 'ApexExecution' OR EventType = 'ApexRestApi' OR EventType = 'ApexSoap' OR EventType = 'ApexTrigger' OR EventType = 'ExternalCustomApexCallout') ORDER BY CreatedDate ASC NULLS FIRST LIMIT 1`),
				},
				Cursor: &cursorConfig{Field: "CreatedDate"},
			},
		})

		probeClient := newLiveSalesforceInput(t, cfg)
		apexELFAvailable, apexELFN, apexELFQ, err := liveProbeApexEventLogFileRowsExist(t, probeClient)
		require.NoError(t, err, "expected Apex EventLogFile probe to succeed")
		if !apexELFAvailable {
			t.Skipf("skipping apex EventLogFile: no historical Apex EventLogFile rows (probe query=%q totalSize=%d).",
				apexELFQ, apexELFN)
		}

		s := newLiveSalesforceInput(t, cfg)

		var first publisher
		first.done = func() {}
		s.publisher = &first
		require.NoError(t, s.RunEventLogFile(), "expected live apex EventLogFile run to succeed")
		firstCursor := s.cursor.EventLogFile.LastEventTime
		require.NotEmpty(t, firstCursor, "expected live apex EventLogFile run to persist last_event_time")
		require.NoError(t, requirePositiveHistoricalLiveRows("apex EventLogFile first run", len(first.published)))

		var second publisher
		second.done = func() {}
		s.publisher = &second
		require.NoError(t, s.RunEventLogFile(), "expected live apex EventLogFile resume run to succeed")
		secondCursor := s.cursor.EventLogFile.LastEventTime
		assertCursorNotBefore(t, firstCursor, secondCursor, "apex EventLogFile last_event_time")

		if len(second.published) > 0 {
			assert.False(t, secondCursor == firstCursor, "apex EventLogFile cursor should advance when a second live run returns more rows")
		}
	})
}
