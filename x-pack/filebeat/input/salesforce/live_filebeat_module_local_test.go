// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build salesforce_live
// +build salesforce_live

// End-to-end Filebeat module smoke for Salesforce (requires a built x-pack filebeat binary).
//
// output.file does not run Elasticsearch ingest node, but the Filebeat module still applies
// its processors (add_fields, etc.), so events include event.dataset / event.provider. Login
// object rows are identified as dataset salesforce.login with provider Object; Apex ELF rows
// are matched via CSV fragments in message (ApexExecution, ...).
//
// Prerequisites:
//   - Build: cd x-pack/filebeat && mage build
//   - Credentials: same as live_validation_local_test.go (SALESFORCE_LIVE_CREDS_FILE or sf-creds.env)
//
// Example:
//   SALESFORCE_LIVE_RUN_FILEBEAT=1 SALESFORCE_LIVE_CREDS_FILE=.../sf-creds.env \
//     go test -tags salesforce_live -timeout 15m -run TestLiveFilebeatSalesforceModuleSmoke -v ./x-pack/filebeat/input/salesforce

package salesforce

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLiveFilebeatSalesforceModuleSmoke(t *testing.T) {
	if !liveFilebeatSmokeEnabled() {
		t.Skipf("set %s=1 to run the real Filebeat Salesforce module smoke", envSalesforceLiveRunFilebeat)
	}

	creds := loadLiveSalesforceCreds(t)
	xpackHome := liveXPackFilebeatHomeDir(t)
	for _, rel := range []string{
		"module/salesforce/login/config/login.yml",
		"module/salesforce/logout/config/logout.yml",
		"module/salesforce/apex/config/apex.yml",
	} {
		p := filepath.Join(xpackHome, filepath.FromSlash(rel))
		_, err := os.Stat(p)
		require.NoError(t, err, "expected shipped Salesforce module input template at %s (path.home must be x-pack/filebeat so Filebeat loads these manifests)", p)
	}
	filebeatBin := liveResolveFilebeatBinary(t, xpackHome)

	apexProbeCfg := liveConfigWithMethod(creds, eventMonitoringMethod{
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
	probeClient := newLiveSalesforceInput(t, apexProbeCfg)
	elfInterval, elfIntervalQuery, elfOK, err := livePickLogFileIntervalForApexELF(t, probeClient)
	require.NoError(t, err, "expected Apex EventLogFile Interval probe to succeed")
	if !elfOK {
		t.Skipf("skipping Filebeat module smoke: no Apex EventLogFile rows for Interval Hourly or Daily (last_query=%q)", elfIntervalQuery)
	}
	t.Logf("module smoke: var.log_file_interval=%s (matched probe query=%q)", elfInterval, elfIntervalQuery)

	// Fresh login so the login fileset can observe LoginEvent on the object path.
	session := generateLoginActivity(t, creds)
	waitCfg := liveConfigWithMethod(creds, eventMonitoringMethod{
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
	waitClient := newLiveSalesforceInput(t, waitCfg)
	_, waitErr := waitForObjectEvent(t, waitClient, "LoginEvent", session.Window, 0)
	require.NoError(t, waitErr, "expected LoginEvent rows after generated login before starting Filebeat")

	tmp := t.TempDir()
	configPath, eventsOutDir := liveWriteFilebeatSalesforceSmokeFiles(t, tmp, creds, elfInterval)

	// --path.home must be the x-pack/filebeat distribution root so Filebeat loads the shipped
	// module manifests under module/salesforce/{login,logout,apex}/config/*.yml (not only inputs in this package).
	liveRunFilebeatTestConfig(t, filebeatBin, xpackHome, configPath)

	// One absolute deadline for both the Filebeat process (context) and the poll loop below,
	// so we never stop waiting for output while the subprocess is still allowed to run.
	smokeDeadline := time.Now().Add(liveFilebeatSmokeMaxRun)
	runCtx, cancel := context.WithDeadline(context.Background(), smokeDeadline)
	defer cancel()

	runErrCh := make(chan struct {
		out []byte
		err error
	}, 1)
	go func() {
		out, err := liveRunFilebeatBounded(runCtx, filebeatBin, xpackHome, configPath)
		runErrCh <- struct {
			out []byte
			err error
		}{out, err}
	}()

	var rows []map[string]interface{}
	var nLogin, nApex int
	loginObserved := false
	lastProgressLog := time.Now()
	for time.Now().Before(smokeDeadline) {
		sz, szErr := liveFilebeatOutputNDJSONSize(eventsOutDir)
		if szErr == nil && sz > 0 {
			parsed, readErr := liveReadFilebeatOutputNDJSONRows(eventsOutDir)
			if readErr == nil {
				rows = parsed
				// Full module wiring adds ECS fields; LoginEvent JSON is in "message" without a "LoginEvent" type string.
				nLogin = liveCountSalesforceModuleDatasetProvider(rows, "salesforce.login", "Object")
				nApex = liveCountLikelyApexELFMessages(rows)
				if nLogin > 0 {
					loginObserved = true
				}
				if loginObserved && nApex > 0 {
					break
				}
			}
		}
		if time.Since(lastProgressLog) > 20*time.Second {
			sz, _ := liveFilebeatOutputNDJSONSize(eventsOutDir)
			paths, _ := liveGlobFilebeatNDJSONOutputFiles(eventsOutDir)
			var parseErr error
			if _, err := liveReadFilebeatOutputNDJSONRows(eventsOutDir); err != nil {
				parseErr = err
			}
			t.Logf("module smoke still waiting: events_rotated_ndjson_size=%d paths=%v login_object_rows=%d apex_elf_rows=%d parse_err=%v", sz, paths, nLogin, nApex, parseErr)
			lastProgressLog = time.Now()
		}
		time.Sleep(750 * time.Millisecond)
	}

	nLogin = liveCountSalesforceModuleDatasetProvider(rows, "salesforce.login", "Object")
	nApex = liveCountLikelyApexELFMessages(rows)
	if nLogin == 0 || nApex == 0 {
		cancel()
		runRes := <-runErrCh
		t.Logf("filebeat output on smoke failure: err=%v", runRes.err)
		if len(runRes.out) > 0 {
			t.Logf("filebeat stdout/stderr (truncated 8k): %s", truncateBytes(runRes.out, 8192))
		}

		sz, _ := liveFilebeatOutputNDJSONSize(eventsOutDir)
		paths, _ := liveGlobFilebeatNDJSONOutputFiles(eventsOutDir)
		t.Logf("diagnostics: events_out_dir=%s rotated_size=%d paths=%v nLogin=%d nApex=%d", eventsOutDir, sz, paths, nLogin, nApex)
		ds := liveCollectUniqueDatasets(rows)
		t.Logf("unique event.dataset values parsed: %v", ds)
		if nLogin == 0 {
			require.FailNow(t, fmt.Sprintf("expected salesforce.login Object rows within %v; got login_object_rows=%d apex_elf_rows=%d", liveFilebeatSmokeMaxRun, nLogin, nApex))
		}
		require.FailNow(t, fmt.Sprintf("expected Apex EventLogFile rows within %v after login rows were observed; got login_object_rows=%d apex_elf_rows=%d", liveFilebeatSmokeMaxRun, nLogin, nApex))
	}

	cancel()

	runRes := <-runErrCh
	if runRes.err != nil && !errors.Is(runRes.err, context.Canceled) && !errors.Is(runRes.err, context.DeadlineExceeded) {
		t.Logf("filebeat run exited with error (often expected after cancel/timeout): %v", runRes.err)
	}

	nLogoutHints := liveCountSalesforceModuleDatasetProvider(rows, "salesforce.logout", "Object")

	require.Greater(t, nLogin, 0, "expected at least one salesforce.login Object row from the shipped module (Real-Time LoginEvent path)")
	require.Greater(t, nApex, 0, "expected at least one Apex EventLogFile CSV row from the shipped module (message heuristics for Apex* EventTypes)")

	t.Logf("module smoke: salesforce.login Object=%d apex ELF (message match)=%d salesforce.logout Object=%d (logout may be zero on this sandbox)", nLogin, nApex, nLogoutHints)
	if nLogoutHints == 0 {
		t.Log("logout fileset: no salesforce.logout Object rows (acceptable on sandboxes where LogoutEvent is not emitted for this path)")
	}
}

func truncateBytes(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}
