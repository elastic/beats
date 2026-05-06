// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build salesforce_live
// +build salesforce_live

// Local-only Salesforce live-test helpers (build tag salesforce_live).
//
// Credential file format: copy sf-creds.env.example to sf-creds.env under
// x-pack/filebeat/input/salesforce/ and populate SALESFORCE_* keys (see example file).

package salesforce

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	envSalesforceLiveGenerateActivity = "SALESFORCE_LIVE_GENERATE_ACTIVITY"
	envSalesforceLiveRunFilebeat      = "SALESFORCE_LIVE_RUN_FILEBEAT"

	// liveFilebeatSmokeMaxRun is the wall-clock budget for the Filebeat subprocess and for the
	// polling loop in TestLiveFilebeatSalesforceModuleSmoke (same absolute deadline). A single
	// value avoids stopping the wait for NDJSON rows while the process can still publish.
	liveFilebeatSmokeMaxRun = 6 * time.Minute

	// liveFilebeatNDJSONMaxLineBytes is the maximum size of one NDJSON line when scanning
	// output.file. bufio.Scanner's default token cap (bufio.MaxScanTokenSize, 64 KiB) fails on
	// large Salesforce payloads embedded in the Beat message field.
	liveFilebeatNDJSONMaxLineBytes = 32 << 20
)

const (
	keySalesforceInstanceURL   = "SALESFORCE_INSTANCE_URL"
	keySalesforceClientID      = "SALESFORCE_CLIENT_ID"
	keySalesforceClientSecret  = "SALESFORCE_CLIENT_SECRET"
	keySalesforceUsername      = "SALESFORCE_USERNAME"
	keySalesforcePassword      = "SALESFORCE_PASSWORD"
	keySalesforceSecurityToken = "SALESFORCE_SECURITY_TOKEN"
	keySalesforceTokenURL      = "SALESFORCE_TOKEN_URL"
)

// Legacy dotenv keys and Instance URL line kept for temporary backward compatibility.
const (
	legacyPrefixInstanceURL = "Instance URL:"
	legacyKeyClientID       = "CLIENT_ID"
	legacyKeyClientSecret   = "CLIENT_SECRET"
	legacyKeyUsername       = "USERNAME"
	legacyKeyPassword       = "PASSWORD"
	legacyKeySecurityToken  = "SECURITY_TOKEN"
	legacyKeyTokenURL       = "TOKEN_URL"
)

func liveActivityGenerationEnabled() bool {
	return os.Getenv(envSalesforceLiveGenerateActivity) == "1"
}

func liveFilebeatSmokeEnabled() bool {
	return os.Getenv(envSalesforceLiveRunFilebeat) == "1"
}

const envFilebeatBinary = "FILEBEAT_BINARY"

// liveXPackFilebeatHomeDir returns the path to x-pack/filebeat (contains module/salesforce).
// It resolves relative to the caller's source file directory (expects .../input/salesforce).
func liveXPackFilebeatHomeDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(1)
	require.True(t, ok, "expected runtime.Caller to succeed")

	dir := filepath.Dir(file)
	home := filepath.Clean(filepath.Join(dir, "..", ".."))
	modDir := filepath.Join(home, "module", "salesforce")
	if _, err := os.Stat(modDir); err != nil {
		require.NoError(t, err, "expected Salesforce module under x-pack/filebeat at %s", modDir)
	}
	return home
}

// liveResolveFilebeatBinary returns FILEBEAT_BINARY when set, otherwise x-pack/filebeat/filebeat
// (or filebeat.exe on Windows). Skips the test when the binary is missing.
func liveResolveFilebeatBinary(t *testing.T, xpackFilebeatHome string) string {
	t.Helper()

	if p := strings.TrimSpace(os.Getenv(envFilebeatBinary)); p != "" {
		_, err := os.Stat(p)
		require.NoError(t, err, "FILEBEAT_BINARY must point to an existing filebeat binary")
		return p
	}

	name := "filebeat"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(xpackFilebeatHome, name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("filebeat binary not found at %s (set %s or build: cd x-pack/filebeat && mage build): %v",
			p, envFilebeatBinary, err)
	}
	return p
}

// livePickLogFileIntervalForApexELF returns Hourly or Daily when that Interval has at least one
// Apex-related EventLogFile row. The shipped apex.yml template filters ELF queries with
// Interval = '{{ .log_file_interval }}'; probing without Interval can succeed while the module
// still sees zero rows if the org only materializes Daily (or vice versa).
// liveModuleUserPasswordTokenURL matches liveConfigWithMethod / the Salesforce input: PasswordCredentials
// expect a base login host; a full .../services/oauth2/token value can make the client request a
// non-existent path (404).
func liveModuleUserPasswordTokenURL(creds liveSalesforceCreds) string {
	return strings.TrimSuffix(strings.TrimSpace(creds.TokenURL), "/services/oauth2/token")
}

func livePickLogFileIntervalForApexELF(t *testing.T, s *salesforceInput) (interval string, lastQuery string, ok bool, err error) {
	t.Helper()
	if s == nil || s.soqlr == nil {
		return "", "", false, fmt.Errorf("livePickLogFileIntervalForApexELF: need salesforceInput with SOQL client")
	}
	var lastAttempt string
	for _, iv := range []string{"Hourly", "Daily"} {
		q := fmt.Sprintf(
			"SELECT Id, CreatedDate, EventType FROM EventLogFile WHERE Interval = '%s' AND (EventType = 'ApexCallout' OR EventType = 'ApexExecution' OR EventType = 'ApexRestApi' OR EventType = 'ApexSoap' OR EventType = 'ApexTrigger' OR EventType = 'ExternalCustomApexCallout') ORDER BY CreatedDate DESC LIMIT 5",
			iv,
		)
		lastAttempt = q
		res, qerr := s.soqlr.Query(&querier{Query: q}, false)
		if qerr != nil {
			return "", q, false, qerr
		}
		if res.TotalSize() > 0 {
			return iv, q, true, nil
		}
	}
	return "", lastAttempt, false, nil
}

// liveWriteFilebeatSalesforceSmokeFiles writes filebeat.yml and modules.d/salesforce.yml under tmpDir
// and returns the main config path and the directory used for output.file (filename events.ndjson).
//
// The libbeat file output uses elastic-agent-libs/file.Rotator with date-based names: the active
// file is events.ndjson-YYYYMMDD.ndjson under that directory, not literally events.ndjson.
// Use liveReadFilebeatOutputNDJSONRows to read parsed events from that directory.
// logFileInterval must match what the org exposes on EventLogFile.Interval (typically Hourly or Daily).
func liveWriteFilebeatSalesforceSmokeFiles(t *testing.T, tmpDir string, creds liveSalesforceCreds, logFileInterval string) (configPath string, eventsOutDir string) {
	t.Helper()

	modulesDir := filepath.Join(tmpDir, "modules.d")
	dataDir := filepath.Join(tmpDir, "data")
	logsDir := filepath.Join(tmpDir, "logs")
	outDir := filepath.Join(tmpDir, "out")

	require.NoError(t, os.MkdirAll(modulesDir, 0o750))
	require.NoError(t, os.MkdirAll(dataDir, 0o750))
	require.NoError(t, os.MkdirAll(logsDir, 0o750))
	require.NoError(t, os.MkdirAll(outDir, 0o750))

	password := creds.Password + creds.SecurityToken
	tokenBase := liveModuleUserPasswordTokenURL(creds)
	moduleYML := fmt.Sprintf(`- module: salesforce
  login:
    enabled: true
    # Match batch.InitialInterval (~lookback) to <=12×object window so the first RunObject can
    # reach "now" in one batched pass (see nextObjectBatchWindow + max_windows_per_run in the module).
    var.initial_interval: 12m
    var.api_version: 56
    var.url: %q
    var.real_time: true
    var.real_time_interval: 1m
    var.event_log_file: true
    var.elf_interval: 1m
    var.log_file_interval: %s
    var.authentication:
      user_password_flow:
        enabled: true
        client.id: %q
        client.secret: %q
        token_url: %q
        username: %q
        password: %q
      jwt_bearer_flow:
        enabled: false

  logout:
    enabled: true
    var.initial_interval: 12m
    var.api_version: 56
    var.url: %q
    var.real_time: true
    var.real_time_interval: 1m
    var.event_log_file: true
    var.elf_interval: 1m
    var.log_file_interval: %s
    var.authentication:
      user_password_flow:
        enabled: true
        client.id: %q
        client.secret: %q
        token_url: %q
        username: %q
        password: %q
      jwt_bearer_flow:
        enabled: false

  apex:
    enabled: true
    # Wider lookback for historical Apex EventLogFile rows; apex fileset has no object batching.
    var.initial_interval: 720h
    var.api_version: 56
    var.url: %q
    var.elf_interval: 1m
    var.log_file_interval: %s
    var.authentication:
      user_password_flow:
        enabled: true
        client.id: %q
        client.secret: %q
        token_url: %q
        username: %q
        password: %q
      jwt_bearer_flow:
        enabled: false
`,
		creds.InstanceURL,
		logFileInterval,
		creds.ClientID, creds.ClientSecret, tokenBase, creds.Username, password,
		creds.InstanceURL,
		logFileInterval,
		creds.ClientID, creds.ClientSecret, tokenBase, creds.Username, password,
		creds.InstanceURL,
		logFileInterval,
		creds.ClientID, creds.ClientSecret, tokenBase, creds.Username, password,
	)

	modulePath := filepath.Join(modulesDir, "salesforce.yml")
	require.NoError(t, os.WriteFile(modulePath, []byte(moduleYML), 0o600))

	eventsOutDir = outDir

	mainYML := fmt.Sprintf(`name: filebeat-salesforce-smoke

filebeat.config.modules:
  path: ${path.config}/modules.d/*.yml
  reload.enabled: false

setup.template.enabled: false
setup.ilm.enabled: false
setup.dashboards.enabled: false

path.config: %q
path.data: %q
path.logs: %q

queue.mem:
  events: 4096
  flush.min_events: 1
  flush.timeout: 1s

output.file:
  path: %q
  filename: events.ndjson

# Send logs to stderr so the smoke test captures diagnostics (avoid only writing to path.logs).
logging.level: info
logging.to_stderr: true
logging.to_files: false
`,
		tmpDir,
		dataDir,
		logsDir,
		outDir,
	)

	configPath = filepath.Join(tmpDir, "filebeat.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(mainYML), 0o600))

	return configPath, eventsOutDir
}

// liveGlobFilebeatNDJSONOutputFiles returns NDJSON output paths from file output.
// Prefer rotated files (events.ndjson-YYYYMMDD.ndjson), but also include events.ndjson
// as a fallback in case rotation naming differs or rotation has not happened yet.
func liveGlobFilebeatNDJSONOutputFiles(eventsOutDir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(eventsOutDir, "events.ndjson-*.ndjson"))
	if err != nil {
		return nil, err
	}

	base := filepath.Join(eventsOutDir, "events.ndjson")
	fi, err := os.Stat(base)
	switch {
	case err == nil && !fi.IsDir():
		paths = append(paths, base)
	case err != nil && !os.IsNotExist(err):
		return nil, err
	}

	return paths, nil
}

// liveFilebeatOutputNDJSONSize returns the total size of rotated NDJSON output files, if any.
func liveFilebeatOutputNDJSONSize(eventsOutDir string) (int64, error) {
	paths, err := liveGlobFilebeatNDJSONOutputFiles(eventsOutDir)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return 0, err
		}
		n += fi.Size()
	}
	return n, nil
}

// liveReadFilebeatOutputNDJSONRows reads all NDJSON event rows from date-rotated file output files.
func liveReadFilebeatOutputNDJSONRows(eventsOutDir string) ([]map[string]interface{}, error) {
	paths, err := liveGlobFilebeatNDJSONOutputFiles(eventsOutDir)
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	for _, p := range paths {
		part, err := liveReadNDJSONFile(p)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

func liveRunFilebeatTestConfig(t *testing.T, filebeatBinary, pathHome, configPath string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, filebeatBinary,
		"test", "config",
		"-c", configPath,
		"-e",
		"--strict.perms=false",
		"--path.home", pathHome,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	require.NoError(t, err, "filebeat test config failed: %s", out.String())
}

// liveRunFilebeatBounded runs filebeat until the context is cancelled or the process exits.
func liveRunFilebeatBounded(ctx context.Context, filebeatBinary, pathHome, configPath string) (stdoutStderr []byte, err error) {
	cmd := exec.CommandContext(ctx, filebeatBinary,
		"-e",
		"--strict.perms=false",
		"-c", configPath,
		"--path.home", pathHome,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.Bytes(), err
}

func liveReadNDJSONFile(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []map[string]interface{}
	s := bufio.NewScanner(f)
	// Override the default 64 KiB max token; large "message" fields exceed it.
	s.Buffer(make([]byte, 0, 64*1024), liveFilebeatNDJSONMaxLineBytes)
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		if len(line) == 0 {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("ndjson decode: %w", err)
		}
		rows = append(rows, m)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func liveEventDatasetFromMap(m map[string]interface{}) string {
	if v, ok := m["event.dataset"].(string); ok && v != "" {
		return v
	}
	ev, ok := m["event"].(map[string]interface{})
	if !ok {
		return ""
	}
	if v, ok := ev["dataset"].(string); ok {
		return v
	}
	return ""
}

// liveEventProviderFromMap returns event.provider when the Beat event is ECS-shaped (nested event map).
func liveEventProviderFromMap(m map[string]interface{}) string {
	if v, ok := m["event.provider"].(string); ok && v != "" {
		return v
	}
	ev, ok := m["event"].(map[string]interface{})
	if !ok {
		return ""
	}
	if v, ok := ev["provider"].(string); ok {
		return v
	}
	return ""
}

// liveCountSalesforceModuleDatasetProvider counts rows from a fully wired Filebeat module run (e.g. output.file)
// where nested event.dataset and event.provider match. provider "" means do not filter on provider.
func liveCountSalesforceModuleDatasetProvider(rows []map[string]interface{}, dataset, provider string) int {
	n := 0
	for _, row := range rows {
		if liveEventDatasetFromMap(row) != dataset {
			continue
		}
		if provider != "" && liveEventProviderFromMap(row) != provider {
			continue
		}
		n++
	}
	return n
}

func liveCountDataset(rows []map[string]interface{}, dataset string) int {
	n := 0
	for _, row := range rows {
		if liveEventDatasetFromMap(row) == dataset {
			n++
		}
	}
	return n
}

// liveMessageFromNDJSONRow returns the Beat "message" field (Salesforce JSON or ELF CSV line).
func liveMessageFromNDJSONRow(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	if msg, ok := m["message"].(string); ok {
		return msg
	}
	return ""
}

// liveCountRowsWithMessageContaining counts NDJSON rows whose message contains sub.
func liveCountRowsWithMessageContaining(rows []map[string]interface{}, sub string) int {
	n := 0
	for _, row := range rows {
		if strings.Contains(liveMessageFromNDJSONRow(row), sub) {
			n++
		}
	}
	return n
}

// liveCountLikelyLoginObjectMessages counts object API rows for LoginEvent (raw JSON in message).
// Used for file-output smokes: Elasticsearch ingest pipelines (which set event.dataset) are not run.
func liveCountLikelyLoginObjectMessages(rows []map[string]interface{}) int {
	n := 0
	for _, row := range rows {
		msg := liveMessageFromNDJSONRow(row)
		if strings.Contains(msg, "LoginEvent") && strings.Contains(msg, "EventDate") {
			n++
		}
	}
	return n
}

// liveCountLikelyApexELFMessages counts EventLogFile CSV rows for Apex event types shipped by the module.
func liveCountLikelyApexELFMessages(rows []map[string]interface{}) int {
	n := 0
	for _, row := range rows {
		// Prefer structured ECS fields when they exist.
		if ds := liveEventDatasetFromMap(row); ds != "" && ds != "salesforce.apex" {
			continue
		}
		if provider := liveEventProviderFromMap(row); provider != "" && provider != "EventLogFile" {
			continue
		}

		msg := liveMessageFromNDJSONRow(row)
		if msg == "" {
			continue
		}
		if strings.Contains(msg, "ApexExecution") ||
			strings.Contains(msg, "ApexCallout") ||
			strings.Contains(msg, "ApexRestApi") ||
			strings.Contains(msg, "ApexSoap") ||
			strings.Contains(msg, "ApexTrigger") ||
			strings.Contains(msg, "ExternalCustomApexCallout") {
			n++
		}
	}
	return n
}

func liveCollectUniqueDatasets(rows []map[string]interface{}) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, row := range rows {
		ds := liveEventDatasetFromMap(row)
		if ds == "" {
			continue
		}
		if _, ok := seen[ds]; !ok {
			seen[ds] = struct{}{}
			out = append(out, ds)
		}
	}
	return out
}

func writeTempCredsFile(t *testing.T, values map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "salesforce.env")

	var lines []string
	for key, value := range values {
		lines = append(lines, key+"="+value)
	}

	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600))
	return path
}

func parseLiveSalesforceCredsFile(path string) (liveSalesforceCreds, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return liveSalesforceCreds{}, err
	}
	return parseLiveSalesforceCredsContent(raw)
}

func parseLiveSalesforceCredsContent(raw []byte) (liveSalesforceCreds, error) {
	var (
		std = make(map[string]string)
		leg = make(map[string]string)

		instanceFromLegacyLine string
	)

	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, legacyPrefixInstanceURL) {
			instanceFromLegacyLine = strings.TrimSpace(strings.TrimPrefix(line, legacyPrefixInstanceURL))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return liveSalesforceCreds{}, fmt.Errorf("invalid Salesforce credential line %q: expected KEY=VALUE", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		switch key {
		case keySalesforceInstanceURL, keySalesforceClientID, keySalesforceClientSecret,
			keySalesforceUsername, keySalesforcePassword, keySalesforceSecurityToken, keySalesforceTokenURL:
			std[key] = value
		case legacyKeyClientID, legacyKeyClientSecret, legacyKeyUsername,
			legacyKeyPassword, legacyKeySecurityToken, legacyKeyTokenURL:
			leg[key] = value
		default:
			// Ignore unknown keys so mixed-format files stay tolerant.
		}
	}

	creds := liveSalesforceCreds{
		InstanceURL:   firstNonEmpty(std[keySalesforceInstanceURL], instanceFromLegacyLine),
		ClientID:      firstNonEmpty(std[keySalesforceClientID], leg[legacyKeyClientID]),
		ClientSecret:  firstNonEmpty(std[keySalesforceClientSecret], leg[legacyKeyClientSecret]),
		Username:      firstNonEmpty(std[keySalesforceUsername], leg[legacyKeyUsername]),
		Password:      firstNonEmpty(std[keySalesforcePassword], leg[legacyKeyPassword]),
		SecurityToken: firstNonEmpty(std[keySalesforceSecurityToken], leg[legacyKeySecurityToken]),
		TokenURL:      firstNonEmpty(std[keySalesforceTokenURL], leg[legacyKeyTokenURL]),
	}

	var missing []string
	addMissing := func(empty bool, standardizedKey string) {
		if empty {
			missing = append(missing, standardizedKey)
		}
	}
	addMissing(creds.InstanceURL == "", keySalesforceInstanceURL)
	addMissing(creds.ClientID == "", keySalesforceClientID)
	addMissing(creds.ClientSecret == "", keySalesforceClientSecret)
	addMissing(creds.Username == "", keySalesforceUsername)
	addMissing(creds.Password == "", keySalesforcePassword)
	addMissing(creds.SecurityToken == "", keySalesforceSecurityToken)
	addMissing(creds.TokenURL == "", keySalesforceTokenURL)

	if len(missing) > 0 {
		return liveSalesforceCreds{}, fmt.Errorf("missing required Salesforce credentials: %s", strings.Join(missing, ", "))
	}

	return creds, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// liveGenerationWindow bounds generated sandbox activity for SOQL correlation.
type liveGenerationWindow struct {
	Start time.Time
	End   time.Time
}

// liveSession captures OAuth session details and the generation window for a login.
type liveSession struct {
	AccessToken string
	InstanceURL string
	Window      liveGenerationWindow
}

// livePollResult holds the last SOQL attempt and row identifiers for diagnostics.
type livePollResult struct {
	LastQuery string
	IDs       []string
	Times     []string
}

const (
	liveObjectPollInterval = 5 * time.Second
	liveObjectPollTimeout  = 2 * time.Minute
	// LogoutEvent can lag OAuth revoke; allow a longer object poll for logout coverage.
	liveLogoutObjectPollTimeout = 5 * time.Minute
	liveELFPollInterval         = 15 * time.Second
	liveELFPollTimeout          = 10 * time.Minute
	liveWindowBuffer            = 2 * time.Minute
)

// waitForCondition polls fn until it returns true, ctx is done, or the optional timeout elapses.
func waitForCondition(ctx context.Context, interval, timeout time.Duration, fn func() (bool, error)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	tick := interval
	if tick <= 0 {
		tick = time.Millisecond
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return waitForConditionDoneErr(ctx.Err(), timeout)
		case <-ticker.C:
		}
	}
}

func waitForConditionDoneErr(err error, timeout time.Duration) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded) && timeout > 0:
		return fmt.Errorf("waitForCondition: timed out after %v: %w", timeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("waitForCondition: context cancelled before condition was met: %w", err)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("waitForCondition: parent context deadline exceeded: %w", err)
	default:
		return fmt.Errorf("waitForCondition: %w", err)
	}
}

func liveTestHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	cfg := defaultConfig()
	c, err := newClient(cfg, context.Background, logp.NewLogger("salesforce_live_helpers"))
	require.NoError(t, err, "expected HTTP client for live helpers")
	return c
}

type oauthPasswordResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	TokenType   string `json:"token_type"`
}

// liveOAuthBaseURL returns the host portion of a Salesforce OAuth token URL.
// SALESFORCE_TOKEN_URL may legitimately be either a base host (e.g.
// "https://login.salesforce.com") or a full token URL ending in
// "/services/oauth2/token"; both are accepted, with surrounding whitespace
// and trailing slashes tolerated.
func liveOAuthBaseURL(rawURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	trimmed = strings.TrimSuffix(trimmed, "/services/oauth2/token")
	return strings.TrimRight(trimmed, "/")
}

func liveOAuthTokenEndpoint(rawURL string) string {
	return liveOAuthBaseURL(rawURL) + "/services/oauth2/token"
}

func liveOAuthRevokeEndpoint(rawURL string) string {
	return liveOAuthBaseURL(rawURL) + "/services/oauth2/revoke"
}

// liveOAuthRevokeTokenBestEffort POSTs to the OAuth2 revoke endpoint after SOAP logout.
// The session may already be invalid; some orgs return 400 — that is non-fatal and logged only.
func liveOAuthRevokeTokenBestEffort(t *testing.T, creds liveSalesforceCreds, accessToken string) {
	t.Helper()

	revokeURL := liveOAuthRevokeEndpoint(creds.TokenURL)
	form := url.Values{}
	form.Set("token", accessToken)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, revokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Logf("salesforce live: OAuth revoke best-effort: build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := liveTestHTTPClient(t)
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("salesforce live: OAuth revoke best-effort: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("salesforce live: OAuth revoke best-effort: read body: %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Logf("salesforce live: OAuth revoke best-effort: status=%s body=%s (ignored after successful SOAP logout)", resp.Status, string(body))
		return
	}
	t.Logf("salesforce live: OAuth revoke best-effort: OK")
}

func TestLiveOAuthRevokeTokenBestEffortIgnoresNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/oauth2/revoke" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_request"}`))
	}))
	t.Cleanup(srv.Close)

	creds := liveSalesforceCreds{TokenURL: srv.URL + "/services/oauth2/token"}
	liveOAuthRevokeTokenBestEffort(t, creds, "dummy-access-token")
}

func liveOAuthPasswordExchange(t *testing.T, creds liveSalesforceCreds) (accessToken, instanceURL string) {
	t.Helper()

	client := liveTestHTTPClient(t)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", creds.Username)
	form.Set("password", creds.Password+creds.SecurityToken)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, liveOAuthTokenEndpoint(creds.TokenURL), strings.NewReader(form.Encode()))
	require.NoError(t, err, "expected OAuth token request to be buildable")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err, "expected OAuth token HTTP request to succeed")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "expected OAuth token response body to be readable")

	if resp.StatusCode != http.StatusOK {
		require.FailNow(t, "OAuth token request failed", "status=%s body=%s", resp.Status, string(body))
	}

	var parsed oauthPasswordResponse
	require.NoError(t, json.Unmarshal(body, &parsed), "expected OAuth token JSON")
	require.NotEmpty(t, parsed.AccessToken, "expected non-empty OAuth access token")
	require.NotEmpty(t, parsed.InstanceURL, "expected non-empty OAuth instance URL")

	return parsed.AccessToken, parsed.InstanceURL
}

func liveEscapeXMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// liveSOAPPartnerLogout ends the API session via Partner SOAP logout(), which
// tends to produce LogoutEvent / logout EventLogFile rows more reliably than
// OAuth token revoke alone (revoke may not emit Real-Time Event Monitoring rows).
func liveSOAPPartnerLogout(t *testing.T, accessToken, instanceURL string) {
	t.Helper()

	client := liveTestHTTPClient(t)
	apiVersion := defaultConfig().Version
	soapURL := fmt.Sprintf("%s/services/Soap/u/%d.0", strings.TrimSuffix(instanceURL, "/"), apiVersion)
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:urn="urn:partner.soap.sforce.com">
<soapenv:Header>
  <urn:SessionHeader>
    <urn:sessionId>%s</urn:sessionId>
  </urn:SessionHeader>
</soapenv:Header>
<soapenv:Body>
  <urn:logout/>
</soapenv:Body>
</soapenv:Envelope>`, liveEscapeXMLText(accessToken))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, soapURL, strings.NewReader(body))
	require.NoError(t, err, "expected SOAP logout request to be buildable")
	req.Header.Set("Content-Type", "text/xml; charset=UTF-8")
	req.Header.Set("SOAPAction", `""`)

	resp, err := client.Do(req)
	require.NoError(t, err, "expected SOAP logout HTTP request to succeed")
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "expected SOAP logout response body to be readable")
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected SOAP logout HTTP 200, got %s body=%s", resp.Status, string(respBody))
	require.NotContains(t, string(respBody), "<faultstring>", "SOAP logout fault: %s", string(respBody))
}

func liveToolingExecuteAnonymousURL(instanceURL string, apiVersion int) string {
	return fmt.Sprintf("%s/services/data/v%d.0/tooling/executeAnonymous", strings.TrimSuffix(instanceURL, "/"), apiVersion)
}

// liveExecuteAnonymousResult is the Tooling REST executeAnonymous JSON payload.
// See https://developer.salesforce.com/docs/atlas.en-us.api_tooling.meta/api_tooling/tooling_api_objects_executeanonymousresult.htm
type liveExecuteAnonymousResult struct {
	Column              int    `json:"column"`
	Line                int    `json:"line"`
	Compiled            bool   `json:"compiled"`
	Success             bool   `json:"success"`
	CompileProblem      string `json:"compileProblem"`
	ExceptionMessage    string `json:"exceptionMessage"`
	ExceptionStackTrace string `json:"exceptionStackTrace"`
}

func liveValidateExecuteAnonymousResponse(statusCode int, body []byte) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("apex executeAnonymous: HTTP %d: %s", statusCode, strings.TrimSpace(string(body)))
	}

	var r liveExecuteAnonymousResult
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("apex executeAnonymous: parsing JSON response: %w; body=%s", err, string(body))
	}

	if !r.Compiled {
		return fmt.Errorf("apex executeAnonymous: compile failed: compileProblem=%q line=%d column=%d (success=%v exceptionMessage=%q)",
			r.CompileProblem, r.Line, r.Column, r.Success, r.ExceptionMessage)
	}
	if !r.Success {
		return fmt.Errorf("apex executeAnonymous: execution failed: exceptionMessage=%q exceptionStackTrace=%q line=%d column=%d",
			r.ExceptionMessage, r.ExceptionStackTrace, r.Line, r.Column)
	}

	return nil
}

func liveExecuteAnonymousApex(t *testing.T, accessToken, instanceURL string) {
	t.Helper()

	client := liveTestHTTPClient(t)
	apiVersion := defaultConfig().Version
	base := liveToolingExecuteAnonymousURL(instanceURL, apiVersion)
	u, err := url.Parse(base)
	require.NoError(t, err, "expected Apex executeAnonymous base URL to parse")
	q := u.Query()
	q.Set("anonymousBody", "System.debug('elastic_salesforce_live_test');")
	u.RawQuery = q.Encode()

	// Tooling executeAnonymous accepts GET with a URL-encoded anonymousBody query
	// parameter on many orgs; POST returns 405 (allowed methods HEAD, GET).
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	require.NoError(t, err, "expected Apex executeAnonymous request to be buildable")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	require.NoError(t, err, "expected Apex executeAnonymous HTTP request to succeed")
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "expected Apex response body to be readable")

	require.NoError(t, liveValidateExecuteAnonymousResponse(resp.StatusCode, respBody),
		"Apex executeAnonymous response invalid (status=%s)", resp.Status)
}

// generateLoginActivity performs a fresh OAuth login and returns session metadata for correlation.
func generateLoginActivity(t *testing.T, creds liveSalesforceCreds) liveSession {
	t.Helper()

	start := time.Now().UTC()
	accessToken, instanceURL := liveOAuthPasswordExchange(t, creds)
	end := time.Now().UTC()

	return liveSession{
		AccessToken: accessToken,
		InstanceURL: instanceURL,
		Window:      liveGenerationWindow{Start: start, End: end},
	}
}

// generateLogoutActivity logs in, ends the session via Partner SOAP logout, and returns the generation window.
func generateLogoutActivity(t *testing.T, creds liveSalesforceCreds) liveGenerationWindow {
	t.Helper()

	session := generateLoginActivity(t, creds)
	start := time.Now().UTC()
	liveSOAPPartnerLogout(t, session.AccessToken, session.InstanceURL)
	liveOAuthRevokeTokenBestEffort(t, creds, session.AccessToken)
	end := time.Now().UTC()

	return liveGenerationWindow{Start: start, End: end}
}

// generateApexActivity executes low-impact anonymous Apex and returns the generation window.
func generateApexActivity(t *testing.T, creds liveSalesforceCreds) liveGenerationWindow {
	t.Helper()

	start := time.Now().UTC()
	accessToken, instanceURL := liveOAuthPasswordExchange(t, creds)
	liveExecuteAnonymousApex(t, accessToken, instanceURL)
	end := time.Now().UTC()

	return liveGenerationWindow{Start: start, End: end}
}

func liveObjectTimeField(objectName string) string {
	switch objectName {
	case "LoginEvent", "LogoutEvent":
		return "EventDate"
	default:
		return "CreatedDate"
	}
}

// requirePositiveGeneratedLiveRows returns an error when a path that relies on
// freshly generated sandbox activity published zero rows.
func requirePositiveGeneratedLiveRows(context string, count int) error {
	if count == 0 {
		return fmt.Errorf("%s: expected at least one published row after generated sandbox activity; got 0", context)
	}
	return nil
}

// requirePositiveHistoricalLiveRows returns an error when a cursor/historical-query
// live path published zero rows (distinct from generated-activity diagnostics).
func requirePositiveHistoricalLiveRows(context string, count int) error {
	if count == 0 {
		return fmt.Errorf("%s: expected at least one published row from the historical query window; got 0", context)
	}
	return nil
}

func TestRequirePositiveGeneratedLiveRowsRejectsEmpty(t *testing.T) {
	err := requirePositiveGeneratedLiveRows("login object batching first run", 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "generated sandbox activity")
}

func TestRequirePositiveHistoricalLiveRowsRejectsEmpty(t *testing.T) {
	err := requirePositiveHistoricalLiveRows("login EventLogFile first run", 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "historical query window")
}

func TestRequirePositiveLiveRowsAcceptsNonZero(t *testing.T) {
	require.NoError(t, requirePositiveGeneratedLiveRows("generated", 3))
	require.NoError(t, requirePositiveHistoricalLiveRows("historical", 2))
}

// liveSOQLDateTimeLiteral formats a time for SOQL DateTime comparisons. Salesforce
// rejects quoted string literals for DateTime fields ("must be of type dateTime and
// should not be enclosed in quotes"); use an unquoted literal instead.
func liveSOQLDateTimeLiteral(t time.Time) string {
	return formatBatchCursorTime(t)
}

// liveProbeLogoutEventRecencyWindow is how far back we look for LogoutEvent rows to
// decide whether Real-Time LogoutEvent data is actively available on the org.
const liveProbeLogoutEventRecencyWindow = 7 * 24 * time.Hour

// liveProbeEventLogFileEventTypeExists runs a minimal SOQL probe matching manual
// EventLogFile checks (Id, CreatedDate, EventType, recent rows). If totalSize is 0,
// that EventType has no historical ELF metadata on this org (skip Login/Logout ELF tests).
// EventLogFile data is hourly and delayed; absence here is not a timing flake in the short term.
func liveProbeEventLogFileEventTypeExists(t *testing.T, s *salesforceInput, eventType string) (exists bool, total int, query string, err error) {
	t.Helper()
	if s == nil || s.soqlr == nil {
		return false, 0, "", fmt.Errorf("liveProbeEventLogFileEventTypeExists: need salesforceInput with SOQL client")
	}
	q := fmt.Sprintf(
		"SELECT Id, CreatedDate, EventType FROM EventLogFile WHERE EventType = '%s' ORDER BY CreatedDate DESC LIMIT 5",
		strings.ReplaceAll(eventType, "'", "''"),
	)
	res, err := s.soqlr.Query(&querier{Query: q}, false)
	if err != nil {
		return false, 0, q, err
	}
	n := res.TotalSize()
	return n > 0, n, q, nil
}

// liveProbeApexEventLogFileRowsExist matches the Apex EventLogFile EventType filter used in
// live_validation_local_test.go; keep the OR list aligned with that SOQL.
func liveProbeApexEventLogFileRowsExist(t *testing.T, s *salesforceInput) (exists bool, total int, query string, err error) {
	t.Helper()
	if s == nil || s.soqlr == nil {
		return false, 0, "", fmt.Errorf("liveProbeApexEventLogFileRowsExist: need salesforceInput with SOQL client")
	}
	q := `SELECT Id, CreatedDate, EventType FROM EventLogFile WHERE (EventType = 'ApexCallout' OR EventType = 'ApexExecution' OR EventType = 'ApexRestApi' OR EventType = 'ApexSoap' OR EventType = 'ApexTrigger' OR EventType = 'ExternalCustomApexCallout') ORDER BY CreatedDate DESC LIMIT 5`
	res, err := s.soqlr.Query(&querier{Query: q}, false)
	if err != nil {
		return false, 0, q, err
	}
	n := res.TotalSize()
	return n > 0, n, q, nil
}

// liveProbeLogoutEventRecencyDiagnostics reports whether any LogoutEvent exists in the
// recency window and the newest EventDate in the org (for skip messaging when API/SOAP
// logout does not produce observable Real-Time LogoutEvent rows).
func liveProbeLogoutEventRecencyDiagnostics(t *testing.T, s *salesforceInput) (recentCount int, newestEventDateInOrg string, recentQuery string, maxEventDateQuery string, err error) {
	t.Helper()
	if s == nil || s.soqlr == nil {
		return 0, "", "", "", fmt.Errorf("liveProbeLogoutEventRecencyDiagnostics: need salesforceInput with SOQL client")
	}
	since := time.Now().UTC().Add(-liveProbeLogoutEventRecencyWindow)
	recentQuery = fmt.Sprintf(
		"SELECT Id, EventDate FROM LogoutEvent WHERE EventDate >= %s ORDER BY EventDate DESC LIMIT 5",
		liveSOQLDateTimeLiteral(since),
	)
	res, err := s.soqlr.Query(&querier{Query: recentQuery}, false)
	if err != nil {
		return 0, "", recentQuery, "", err
	}
	recentCount = res.TotalSize()

	maxEventDateQuery = "SELECT EventDate FROM LogoutEvent ORDER BY EventDate DESC LIMIT 1"
	res2, err := s.soqlr.Query(&querier{Query: maxEventDateQuery}, false)
	if err != nil {
		return recentCount, "", recentQuery, maxEventDateQuery, err
	}
	if res2.TotalSize() > 0 {
		fields := res2.Records()[0].Record().Fields()
		if ts, ok := fields["EventDate"].(string); ok {
			newestEventDateInOrg = ts
		}
	}
	return recentCount, newestEventDateInOrg, recentQuery, maxEventDateQuery, nil
}

// waitForObjectEvent polls SOQL until rows appear in the buffered window or the object poll budget expires.
// On failure (including timeout or repeated SOQL errors), it returns a non-nil error and the last poll snapshot for diagnostics.
func waitForObjectEvent(t *testing.T, s *salesforceInput, objectName string, window liveGenerationWindow, pollTimeout time.Duration) (livePollResult, error) {
	t.Helper()

	if s == nil || s.soqlr == nil {
		return livePollResult{}, fmt.Errorf("waitForObjectEvent: need salesforceInput with an active SOQL client")
	}

	if pollTimeout <= 0 {
		pollTimeout = liveObjectPollTimeout
	}

	timeField := liveObjectTimeField(objectName)
	startBuf := window.Start.Add(-liveWindowBuffer).UTC()

	var result livePollResult
	pollErr := waitForCondition(context.Background(), liveObjectPollInterval, pollTimeout, func() (bool, error) {
		// Slide the upper bound forward on each attempt so late-indexed events are not
		// excluded by a fixed window_end captured at poll start.
		endBuf := time.Now().UTC().Add(liveWindowBuffer)
		q := fmt.Sprintf(
			"SELECT Id, %s FROM %s WHERE %s >= %s AND %s <= %s ORDER BY %s DESC LIMIT 50",
			timeField,
			objectName,
			timeField,
			liveSOQLDateTimeLiteral(startBuf),
			timeField,
			liveSOQLDateTimeLiteral(endBuf),
			timeField,
		)
		result.LastQuery = q

		res, err := s.soqlr.Query(&querier{Query: q}, false)
		if err != nil {
			return false, err
		}

		ids := make([]string, 0, res.TotalSize())
		times := make([]string, 0, res.TotalSize())
		for _, rec := range res.Records() {
			fields := rec.Record().Fields()
			if id, ok := fields["Id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
			if ts, ok := fields[timeField].(string); ok && ts != "" {
				times = append(times, ts)
			}
		}
		result.IDs = ids
		result.Times = times

		return len(ids) > 0, nil
	})

	if pollErr != nil {
		return result, fmt.Errorf("waitForObjectEvent: object=%s window_start=%s window_end=%s last_query=%q partial_ids=%d: %w",
			objectName, window.Start.UTC(), window.End.UTC(), result.LastQuery, len(result.IDs), pollErr)
	}

	return result, nil
}

// waitForEventLogFile polls EventLogFile rows for eventType within the buffered window or the ELF poll budget expires.
// On failure (including timeout or repeated SOQL errors), it returns a non-nil error and the last poll snapshot for diagnostics.
func waitForEventLogFile(t *testing.T, s *salesforceInput, eventType string, window liveGenerationWindow, pollTimeout time.Duration) (livePollResult, error) {
	t.Helper()

	if s == nil || s.soqlr == nil {
		return livePollResult{}, fmt.Errorf("waitForEventLogFile: need salesforceInput with an active SOQL client")
	}

	if pollTimeout <= 0 {
		pollTimeout = liveELFPollTimeout
	}

	startBuf := window.Start.Add(-liveWindowBuffer).UTC()

	var result livePollResult
	pollErr := waitForCondition(context.Background(), liveELFPollInterval, pollTimeout, func() (bool, error) {
		endBuf := time.Now().UTC().Add(liveWindowBuffer)
		q := fmt.Sprintf(
			"SELECT Id, CreatedDate, LogDate, EventType FROM EventLogFile WHERE EventType = '%s' AND CreatedDate >= %s AND CreatedDate <= %s ORDER BY CreatedDate DESC LIMIT 50",
			strings.ReplaceAll(eventType, "'", "''"),
			liveSOQLDateTimeLiteral(startBuf),
			liveSOQLDateTimeLiteral(endBuf),
		)
		result.LastQuery = q

		res, err := s.soqlr.Query(&querier{Query: q}, false)
		if err != nil {
			return false, err
		}

		ids := make([]string, 0, res.TotalSize())
		times := make([]string, 0, res.TotalSize())
		for _, rec := range res.Records() {
			fields := rec.Record().Fields()
			if id, ok := fields["Id"].(string); ok && id != "" {
				ids = append(ids, id)
			}
			if ts, ok := fields["CreatedDate"].(string); ok && ts != "" {
				times = append(times, ts)
			}
		}
		result.IDs = ids
		result.Times = times

		return len(ids) > 0, nil
	})

	if pollErr != nil {
		return result, fmt.Errorf("waitForEventLogFile: event_type=%s window_start=%s window_end=%s last_query=%q partial_ids=%d: %w",
			eventType, window.Start.UTC(), window.End.UTC(), result.LastQuery, len(result.IDs), pollErr)
	}

	return result, nil
}

func TestWaitForConditionReturnsBeforeTimeout(t *testing.T) {
	attempts := 0
	err := waitForCondition(context.Background(), 10*time.Millisecond, time.Second, func() (bool, error) {
		attempts++
		return attempts == 3, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestWaitForConditionReturnsTimeoutContext(t *testing.T) {
	err := waitForCondition(context.Background(), 5*time.Millisecond, 20*time.Millisecond, func() (bool, error) {
		return false, nil
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out")
}

func TestWaitForConditionParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForCondition(parent, 5*time.Millisecond, time.Minute, func() (bool, error) {
		return false, nil
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "cancelled")
	assert.NotContains(t, err.Error(), "timed out")
}

func TestLiveValidateExecuteAnonymousResponse(t *testing.T) {
	t.Run("accepts successful tooling response", func(t *testing.T) {
		body := `{"line":-1,"column":-1,"compiled":true,"compileProblem":null,"success":true,"exceptionMessage":"","exceptionStackTrace":""}`
		require.NoError(t, liveValidateExecuteAnonymousResponse(http.StatusOK, []byte(body)))
	})

	t.Run("rejects non-200", func(t *testing.T) {
		err := liveValidateExecuteAnonymousResponse(http.StatusServiceUnavailable, []byte("upstream"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "503")
		assert.ErrorContains(t, err, "upstream")
	})

	t.Run("rejects compile failure", func(t *testing.T) {
		body := `{"compiled":false,"success":false,"compileProblem":"unexpected token","line":1,"column":2}`
		err := liveValidateExecuteAnonymousResponse(http.StatusOK, []byte(body))
		require.Error(t, err)
		assert.ErrorContains(t, err, "compile")
		assert.ErrorContains(t, err, "unexpected token")
	})

	t.Run("rejects runtime exception", func(t *testing.T) {
		body := `{"compiled":true,"success":false,"exceptionMessage":"Divide by 0","exceptionStackTrace":"line 1"}`
		err := liveValidateExecuteAnonymousResponse(http.StatusOK, []byte(body))
		require.Error(t, err)
		assert.ErrorContains(t, err, "Divide by 0")
		assert.ErrorContains(t, err, "execution failed")
	})

	t.Run("rejects invalid JSON on 200", func(t *testing.T) {
		err := liveValidateExecuteAnonymousResponse(http.StatusOK, []byte(`not json`))
		require.Error(t, err)
		assert.ErrorContains(t, err, "parsing JSON")
	})
}

func TestWaitForObjectEventRequiresSOQLClient(t *testing.T) {
	_, err := waitForObjectEvent(t, nil, "LoginEvent", liveGenerationWindow{}, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "SOQL client")
}

func TestWaitForEventLogFileRequiresSOQLClient(t *testing.T) {
	_, err := waitForEventLogFile(t, nil, "Login", liveGenerationWindow{}, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "SOQL client")
}

func TestLoadLiveSalesforceCredsPrefersStandardizedKeys(t *testing.T) {
	path := writeTempCredsFile(t, map[string]string{
		keySalesforceInstanceURL:   "https://example.my.salesforce.com",
		keySalesforceClientID:      "cid",
		keySalesforceClientSecret:  "secret",
		keySalesforceUsername:      "user@example.com",
		keySalesforcePassword:      "password",
		keySalesforceSecurityToken: "token",
		keySalesforceTokenURL:      "https://test.salesforce.com/services/oauth2/token",
	})

	creds, err := parseLiveSalesforceCredsFile(path)
	require.NoError(t, err)

	assert.Equal(t, "https://example.my.salesforce.com", creds.InstanceURL)
	assert.Equal(t, "cid", creds.ClientID)
	assert.Equal(t, "secret", creds.ClientSecret)
	assert.Equal(t, "user@example.com", creds.Username)
	assert.Equal(t, "password", creds.Password)
	assert.Equal(t, "token", creds.SecurityToken)
	assert.Equal(t, "https://test.salesforce.com/services/oauth2/token", creds.TokenURL)
}

func TestLoadLiveSalesforceCredsPrefersStandardizedKeysOverLegacy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "salesforce.env")
	content := strings.Join([]string{
		legacyPrefixInstanceURL + " https://legacy-line.example.com",
		"SALESFORCE_INSTANCE_URL=https://std.example.com",
		"SALESFORCE_CLIENT_ID=std-id",
		"CLIENT_ID=legacy-id",
		"SALESFORCE_CLIENT_SECRET=std-secret",
		"CLIENT_SECRET=legacy-secret",
		"SALESFORCE_USERNAME=std@example.com",
		"USERNAME=legacy@example.com",
		"SALESFORCE_PASSWORD=std-pass",
		"PASSWORD=legacy-pass",
		"SALESFORCE_SECURITY_TOKEN=std-tok",
		"SECURITY_TOKEN=legacy-tok",
		"SALESFORCE_TOKEN_URL=https://std.example.com/services/oauth2/token",
		"TOKEN_URL=https://legacy.example.com/services/oauth2/token",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	creds, err := parseLiveSalesforceCredsFile(path)
	require.NoError(t, err)

	assert.Equal(t, "https://std.example.com", creds.InstanceURL)
	assert.Equal(t, "std-id", creds.ClientID)
	assert.Equal(t, "std-secret", creds.ClientSecret)
	assert.Equal(t, "std@example.com", creds.Username)
	assert.Equal(t, "std-pass", creds.Password)
	assert.Equal(t, "std-tok", creds.SecurityToken)
	assert.Equal(t, "https://std.example.com/services/oauth2/token", creds.TokenURL)
}

func TestLoadLiveSalesforceCredsRejectsMissingStandardizedKeys(t *testing.T) {
	path := writeTempCredsFile(t, map[string]string{
		keySalesforceInstanceURL: "https://example.my.salesforce.com",
	})

	_, err := parseLiveSalesforceCredsFile(path)
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing required Salesforce credentials:")
	assert.ErrorContains(t, err, keySalesforceClientID)
	assert.ErrorContains(t, err, keySalesforceClientSecret)
	assert.ErrorContains(t, err, keySalesforceUsername)
	assert.ErrorContains(t, err, keySalesforcePassword)
	assert.ErrorContains(t, err, keySalesforceSecurityToken)
	assert.ErrorContains(t, err, keySalesforceTokenURL)
	assert.NotContains(t, err.Error(), keySalesforceInstanceURL, "instance URL was provided and should not be listed as missing")
}

func TestLoadLiveSalesforceCredsAcceptsLegacyKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "salesforce.env")
	content := strings.Join([]string{
		legacyPrefixInstanceURL + " https://legacy.example.com",
		"CLIENT_ID=cid",
		"CLIENT_SECRET=secret",
		"USERNAME=user@example.com",
		"PASSWORD=password",
		"SECURITY_TOKEN=token",
		"TOKEN_URL=https://test.salesforce.com/services/oauth2/token",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	creds, err := parseLiveSalesforceCredsFile(path)
	require.NoError(t, err)

	assert.Equal(t, "https://legacy.example.com", creds.InstanceURL)
	assert.Equal(t, "cid", creds.ClientID)
	assert.Equal(t, "secret", creds.ClientSecret)
	assert.Equal(t, "user@example.com", creds.Username)
	assert.Equal(t, "password", creds.Password)
	assert.Equal(t, "token", creds.SecurityToken)
	assert.Equal(t, "https://test.salesforce.com/services/oauth2/token", creds.TokenURL)
}

func TestLoadLiveSalesforceCredsAcceptsQuotedValues(t *testing.T) {
	path := writeTempCredsFile(t, map[string]string{
		keySalesforceInstanceURL:   `"https://example.my.salesforce.com"`,
		keySalesforceClientID:      `"cid"`,
		keySalesforceClientSecret:  `"secret"`,
		keySalesforceUsername:      "'user@example.com'",
		keySalesforcePassword:      `"password"`,
		keySalesforceSecurityToken: "'token'",
		keySalesforceTokenURL:      `"https://test.salesforce.com/services/oauth2/token"`,
	})

	creds, err := parseLiveSalesforceCredsFile(path)
	require.NoError(t, err)

	assert.Equal(t, "https://example.my.salesforce.com", creds.InstanceURL)
	assert.Equal(t, "cid", creds.ClientID)
	assert.Equal(t, "secret", creds.ClientSecret)
	assert.Equal(t, "user@example.com", creds.Username)
	assert.Equal(t, "password", creds.Password)
	assert.Equal(t, "token", creds.SecurityToken)
	assert.Equal(t, "https://test.salesforce.com/services/oauth2/token", creds.TokenURL)
}

func TestLiveGlobFilebeatNDJSONOutputFilesIncludesBaseAndRotated(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "events.ndjson")
	rotated := filepath.Join(dir, "events.ndjson-20260416.ndjson")
	require.NoError(t, os.WriteFile(base, []byte("{}\n"), 0o600))
	require.NoError(t, os.WriteFile(rotated, []byte("{}\n"), 0o600))

	paths, err := liveGlobFilebeatNDJSONOutputFiles(dir)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{base, rotated}, paths)
}

func TestLiveModuleUserPasswordTokenURLTrimsOAuthPath(t *testing.T) {
	c := liveSalesforceCreds{TokenURL: "https://login.salesforce.com/services/oauth2/token"}
	assert.Equal(t, "https://login.salesforce.com", liveModuleUserPasswordTokenURL(c))
}

func TestLiveActivityAndFilebeatEnvGates(t *testing.T) {
	t.Setenv(envSalesforceLiveGenerateActivity, "")
	t.Setenv(envSalesforceLiveRunFilebeat, "")
	assert.False(t, liveActivityGenerationEnabled())
	assert.False(t, liveFilebeatSmokeEnabled())

	t.Setenv(envSalesforceLiveGenerateActivity, "1")
	t.Setenv(envSalesforceLiveRunFilebeat, "0")
	assert.True(t, liveActivityGenerationEnabled())
	assert.False(t, liveFilebeatSmokeEnabled())

	t.Setenv(envSalesforceLiveGenerateActivity, "0")
	t.Setenv(envSalesforceLiveRunFilebeat, "1")
	assert.False(t, liveActivityGenerationEnabled())
	assert.True(t, liveFilebeatSmokeEnabled())
}

func TestLiveOAuthTokenEndpointAcceptsBothFormats(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"base host", "https://login.salesforce.com", "https://login.salesforce.com/services/oauth2/token"},
		{"base host trailing slash", "https://login.salesforce.com/", "https://login.salesforce.com/services/oauth2/token"},
		{"full token url", "https://login.salesforce.com/services/oauth2/token", "https://login.salesforce.com/services/oauth2/token"},
		{"full token url trailing slash", "https://login.salesforce.com/services/oauth2/token/", "https://login.salesforce.com/services/oauth2/token"},
		{"surrounding whitespace", "  https://login.salesforce.com  ", "https://login.salesforce.com/services/oauth2/token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, liveOAuthTokenEndpoint(tc.in))
		})
	}
}

func TestLiveOAuthRevokeEndpointAcceptsBothFormats(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"base host", "https://login.salesforce.com", "https://login.salesforce.com/services/oauth2/revoke"},
		{"base host trailing slash", "https://login.salesforce.com/", "https://login.salesforce.com/services/oauth2/revoke"},
		{"full token url", "https://login.salesforce.com/services/oauth2/token", "https://login.salesforce.com/services/oauth2/revoke"},
		{"full token url trailing slash", "https://login.salesforce.com/services/oauth2/token/", "https://login.salesforce.com/services/oauth2/revoke"},
		{"surrounding whitespace", "  https://login.salesforce.com  ", "https://login.salesforce.com/services/oauth2/revoke"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, liveOAuthRevokeEndpoint(tc.in))
		})
	}
}

func TestLiveOAuthPasswordExchangeAcceptsBaseHost(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/oauth2/token" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok","instance_url":"https://instance.example.test","token_type":"Bearer"}`)
	}))
	t.Cleanup(srv.Close)

	creds := liveSalesforceCreds{
		TokenURL:      srv.URL,
		Username:      "u",
		Password:      "p",
		SecurityToken: "s",
		ClientID:      "cid",
		ClientSecret:  "cs",
	}
	accessToken, instanceURL := liveOAuthPasswordExchange(t, creds)
	assert.Equal(t, "tok", accessToken)
	assert.Equal(t, "https://instance.example.test", instanceURL)
	assert.Equal(t, 1, hits)
}

func TestLiveOAuthPasswordExchangeAcceptsFullTokenURL(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/oauth2/token" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok","instance_url":"https://instance.example.test","token_type":"Bearer"}`)
	}))
	t.Cleanup(srv.Close)

	creds := liveSalesforceCreds{
		TokenURL:      srv.URL + "/services/oauth2/token",
		Username:      "u",
		Password:      "p",
		SecurityToken: "s",
		ClientID:      "cid",
		ClientSecret:  "cs",
	}
	accessToken, instanceURL := liveOAuthPasswordExchange(t, creds)
	assert.Equal(t, "tok", accessToken)
	assert.Equal(t, "https://instance.example.test", instanceURL)
	assert.Equal(t, 1, hits)
}

func TestLiveOAuthRevokeTokenBestEffortAcceptsBaseHost(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/oauth2/revoke" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	creds := liveSalesforceCreds{TokenURL: srv.URL}
	liveOAuthRevokeTokenBestEffort(t, creds, "dummy-access-token")
	assert.Equal(t, 1, hits)
}
