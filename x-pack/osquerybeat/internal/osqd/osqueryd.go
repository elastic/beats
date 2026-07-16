// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/common/proc"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqlog"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	osqueryDName               = "osqueryd"
	osqueryDarwinAppBundlePath = "osquery.app/Contents/MacOS"
)

const (
	defaultDataDir               = "osquery"
	defaultCertsDir              = "certs"
	defaultLensesDir             = "lenses"
	defaultConfigRefreshInterval = 30 // interval osqueryd will poll for configuration changed; scheduled queries configuration for now
)

const (
	flagEnableTables  = "enable_tables"
	flagDisableTables = "disable_tables"
)

var (
	defaultDisabledTables = []string{"carves", "curl"}
)

type Runner interface {
	Check(ctx context.Context) error
	Run(ctx context.Context, flags Flags) error
	SocketPath() string
	DataPath() string
	// SetExtensions configures customer-managed extension entries (directories,
	// files, or glob patterns) to resolve and autoload, an optional
	// extensions_timeout override (seconds, reset to the default when <= 0), and an
	// optional list of extension names osqueryd must wait for at startup
	// (extensions_require). It must be called before Run so prepare() writes the
	// autoload file for the desired set on (re)start.
	SetExtensions(paths []string, timeout int, require []string)
}

type RunnerFactory func(socketPath string, opts ...Option) (Runner, error)

type OSQueryD struct {
	socketPath string
	binPath    string
	extPath    string
	dataPath   string
	certsPath  string
	lensesPath string

	configPlugin string
	loggerPlugin string

	extensionsTimeout     int
	configRefreshInterval int

	// baseExtensionsTimeout is the effective extensions_timeout after construction
	// options are applied; SetExtensions reverts to it when the configuration no
	// longer overrides the timeout.
	baseExtensionsTimeout int

	// extensionEntries holds absolute directories, files, or glob patterns that
	// are resolved into customer-managed extension binaries appended to the
	// autoload file after the mandatory Elastic extension. extensionRequire holds
	// extension names osqueryd must wait for at startup (extensions_require).
	extMx            sync.Mutex
	extensionEntries []string
	extensionRequire []string

	log *logp.Logger
}

type Option func(*OSQueryD)

func WithExtensionsTimeout(to int) Option {
	return func(q *OSQueryD) {
		q.extensionsTimeout = to
	}
}

func WithBinaryPath(binPath string) Option {
	return func(q *OSQueryD) {
		q.binPath = binPath
	}
}

func WithExtensionPath(extPath string) Option {
	return func(q *OSQueryD) {
		q.extPath = extPath
	}
}

// WithExtensions sets the initial customer-managed extension entries (directories,
// files, or glob patterns) to resolve.
func WithExtensions(paths []string) Option {
	return func(q *OSQueryD) {
		q.extensionEntries = append([]string(nil), paths...)
	}
}

func WithConfigRefresh(refreshInterval int) Option {
	return func(q *OSQueryD) {
		q.configRefreshInterval = refreshInterval
	}
}

func WithDataPath(dataPath string) Option {
	return func(q *OSQueryD) {
		q.dataPath = dataPath
	}
}

func WithLogger(log *logp.Logger) Option {
	return func(q *OSQueryD) {
		q.log = log
	}
}

func WithConfigPlugin(name string) Option {
	return func(q *OSQueryD) {
		q.configPlugin = name
	}
}

func WithLoggerPlugin(name string) Option {
	return func(q *OSQueryD) {
		q.loggerPlugin = name
	}
}

func New(socketPath string, opts ...Option) (Runner, error) {
	return newOsqueryD(socketPath, opts...)
}

func newOsqueryD(socketPath string, opts ...Option) (*OSQueryD, error) {
	q := &OSQueryD{
		socketPath:            socketPath,
		extensionsTimeout:     defaultExtensionsTimeout,
		configRefreshInterval: defaultConfigRefreshInterval,
	}

	for _, opt := range opts {
		opt(q)
	}

	// Remember the post-options timeout so SetExtensions can revert to it when the
	// configuration override is removed.
	q.baseExtensionsTimeout = q.extensionsTimeout

	// The working directory is set to something like ./data/elastic-agent-3afa07/run/osquery-default by the agent
	// Use the child dir osquery for that, so the full path is resolved to ./data/elastic-agent-3afa07/run/osquery-default/oquery
	//
	// The following files are currently created there by osqueryd executable when it is started
	//
	// -rw-------   1 root  wheel  149 Nov 28 17:46 osquery.autoload
	// drwx------  11 root  wheel  352 Nov 28 19:00 osquery.db
	// -rw-r--r--   1 root  wheel    0 Nov 28 17:46 osquery.flags
	// -rw-------   1 root  wheel    5 Nov 28 18:48 osquery.pid
	if q.dataPath == "" {
		q.dataPath = defaultDataDir
	}

	// Initialize binPath before certsPath and the lensesPath are set
	err := q.prepareBinPath()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bin path, %w", err)
	}

	if q.certsPath == "" {
		q.certsPath = filepath.Join(q.binPath, defaultCertsDir)
	}

	if q.lensesPath == "" {
		q.lensesPath = filepath.Join(q.binPath, defaultLensesDir)
	}

	return q, nil
}

func (q *OSQueryD) SocketPath() string {
	return q.socketPath
}

func (q *OSQueryD) DataPath() string {
	return q.dataPath
}

// SetExtensions updates the customer-managed extension entries (directories, files,
// or glob patterns) to resolve, an optional extensions_timeout override (seconds,
// reverts to the construction-time default when <= 0), and the extension names
// osqueryd must wait for at startup (extensions_require). The new set is applied on
// the next Run (which rewrites the autoload file via prepare()).
func (q *OSQueryD) SetExtensions(paths []string, timeout int, require []string) {
	q.extMx.Lock()
	defer q.extMx.Unlock()
	q.extensionEntries = append([]string(nil), paths...)
	q.extensionRequire = append([]string(nil), require...)
	if timeout > 0 {
		q.extensionsTimeout = timeout
	} else {
		q.extensionsTimeout = q.baseExtensionsTimeout
	}
}

func (q *OSQueryD) getExtensionEntries() []string {
	q.extMx.Lock()
	defer q.extMx.Unlock()
	return append([]string(nil), q.extensionEntries...)
}

func (q *OSQueryD) getExtensionRequire() []string {
	q.extMx.Lock()
	defer q.extMx.Unlock()
	return append([]string(nil), q.extensionRequire...)
}

func (q *OSQueryD) getExtensionsTimeout() int {
	q.extMx.Lock()
	defer q.extMx.Unlock()
	return q.extensionsTimeout
}

// AutoloadPath returns the path of the osquery extensions autoload file within
// the given osquery data directory.
func AutoloadPath(dataPath string) string {
	return filepath.Join(dataPath, osqueryAutoload)
}

// Check checks if the binary exists and executable
func (q *OSQueryD) Check(ctx context.Context) error {
	err := q.prepareBinPath()
	if err != nil {
		return fmt.Errorf("failed to prepare bin path, %w", err)
	}

	//nolint:gosec // works as expected
	cmd := exec.CommandContext(
		ctx,
		osquerydPath(q.binPath),
		"--S",
		"--version",
	)

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

// Run executes osqueryd binary as a child process
func (q *OSQueryD) Run(ctx context.Context, flags Flags) error {
	cleanup, err := q.prepare()
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := q.createCommand(ctx, flags)

	q.log.Debugf("start osqueryd process: args: %v", cmd.Args)

	cmd.SysProcAttr = setpgid()

	// Read standard output
	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = q.logOSQueryOutput(ctx, stdout)
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Assign osqueryd process to the JobObject on windows
	// in order to assure no orphan process is left behind
	// after osquerybeat process is killed.
	if err := proc.JobObject.Assign(cmd.Process); err != nil {
		q.log.Errorf("osqueryd process failed job assign: %v", err)
	}

	// Capture stderr for error messages
	// Log stderr line-by-line at error level for better visibility
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = q.logOSQueryOutput(ctx, stderr)
	}()

	finished := make(chan error, 1)

	// Wait on osqueryd exit
	wg.Add(1)
	go func() {
		defer wg.Done()
		finished <- cmd.Wait()
	}()

	select {
	case err = <-finished:
		if err != nil {
			q.log.Errorf("osqueryd process exited with error: %v", err)
		} else {
			q.log.Info("osqueryd process exited")
		}
	case <-ctx.Done():
		q.log.Debug("kill process group on context done")
		if err := killProcessGroup(cmd); err != nil {
			q.log.Errorf("kill process group failed: %v", err)
		}
		// Wait till finished
		<-finished
	}

	wg.Wait()

	return err
}

func (q *OSQueryD) prepare() (func(), error) {
	err := q.prepareBinPath()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bin path, %w", err)
	}

	// Create data directory for all the osquery config/runtime files
	if err := os.MkdirAll(q.dataPath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create dir %v, %w", q.dataPath, err)
	}

	// If socket path was not specified, create
	if q.socketPath == "" {
		// Create temp directory for socket and possibly other things
		// The unix domain socker path is limited to 108 chars and would
		// not always be able to create in subdirectory
		socketPath, cleanupFn, err := CreateSocketPath()
		if err != nil {
			return nil, err
		}
		q.socketPath = socketPath
		return cleanupFn, nil
	}

	// Prepare autoload osquery-extension
	extensionPath := q.extPath
	if extensionPath == "" {
		extensionPath = osqueryExtensionPath(q.binPath)
	}
	if _, err := os.Stat(extensionPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("extension path does not exist: %s, %w", extensionPath, err)
		} else {
			return nil, fmt.Errorf("failed to stat extension path, %w", err)
		}
	}

	// Resolve the configured entries (directories, files, or globs) into
	// customer-managed extension binaries. Invalid entries are logged and skipped
	// so a bad extension never aborts osqueryd startup.
	extraExtensions := q.collectExtensionBinaries(q.getExtensionEntries())

	// Write the autoload file
	extensionAutoloadPath := q.resolveDataPath(osqueryAutoload)
	err = prepareAutoloadFile(extensionAutoloadPath, extensionPath, extraExtensions, q.log)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare extensions autoload file, %w", err)
	}

	// Write the flagsfile in order to lock down/prevent loading default flags from osquery global locations.
	// Otherwise the osqueryi and osqueryd will try to load the default flags file,
	// for example from /var/osquery/osquery.flags.default on Mac, and can potentially mess up configuration of our osquery instance.
	flagsfilePath := q.resolveDataPath(osqueryFlagfile)
	exists, err := fileutil.FileExists(flagsfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check flagsfile path, %w", err)
	}
	if !exists {
		f, err := os.OpenFile(flagsfilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create flagsfile, %w", err)
		}
		f.Close()
	}

	return func() {}, nil
}

func prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath string, extraPaths []string, log *logp.Logger) error {
	desired := autoloadContent(mandatoryExtensionPath, extraPaths)

	rewrite := true
	existing, err := os.ReadFile(extensionAutoloadPath)
	switch {
	case err == nil:
		// Rewrite only when the desired content differs. The desired content is
		// freshly computed from the resolved extension set (line 0 is the mandatory
		// extension by construction), so when it matches there is nothing a rewrite
		// could fix.
		if string(existing) == desired {
			log.Debugf("Extensions autoload file %s is up to date", extensionAutoloadPath)
			rewrite = false
		} else {
			log.Debugf("Extensions autoload file %s differs from desired content, rewrite it", extensionAutoloadPath)
		}
	case os.IsNotExist(err):
		log.Debugf("Extensions autoload file %s doesn't exists, create a new one", extensionAutoloadPath)
	default:
		return fmt.Errorf("failed to read osquery.autoload file, %w", err)
	}

	if rewrite {
		if err := os.WriteFile(extensionAutoloadPath, []byte(desired), 0600); err != nil {
			return fmt.Errorf("failed write osquery extension autoload file, %w", err)
		}
	}
	return nil
}

// autoloadContent builds the desired osquery.autoload file content: the mandatory
// Elastic extension always on the first line, followed by any customer-managed
// extension paths (deduplicated, mandatory excluded).
func autoloadContent(mandatoryExtensionPath string, extraPaths []string) string {
	lines := []string{mandatoryExtensionPath}
	seen := map[string]struct{}{mandatoryExtensionPath: {}}
	for _, p := range extraPaths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		lines = append(lines, p)
	}
	return strings.Join(lines, "\n")
}

// collectExtensionBinaries resolves the configured entries, logs the outcome, and
// returns the valid extension binary paths safe to reference in the autoload file.
func (q *OSQueryD) collectExtensionBinaries(entries []string) []string {
	results := ResolveExtensions(entries)
	var out []string
	for _, res := range results {
		if res.Error != "" {
			if q.log != nil {
				q.log.Warnf("Skipping customer-managed osquery extension entry %q: %v. Custom extensions are not developed, validated, or supported by Elastic.", res.Entry, res.Error)
			}
			continue
		}
		for _, skip := range res.Skipped {
			if q.log != nil {
				q.log.Warnf("Skipping customer-managed osquery extension %q: %v. Ensure the binary is a regular executable file owned by the osquery user and not writable by group or others.", skip.Path, skip.Reason)
			}
		}
		for _, p := range res.Loaded {
			if q.log != nil {
				q.log.Infof("Autoloading customer-managed osquery extension %q (unsupported by Elastic; loaded at customer's own risk)", p)
			}
			out = append(out, p)
		}
	}
	return out
}

// ExtensionSkip records a customer-managed extension binary that was skipped and why.
type ExtensionSkip struct {
	Path   string
	Reason string
}

// ExtensionResolveResult holds the outcome of resolving a single configured entry
// (a directory, a file, or a glob pattern).
type ExtensionResolveResult struct {
	Entry   string
	Error   string // entry-level error (e.g. not an absolute path, missing file, bad glob)
	Loaded  []string
	Skipped []ExtensionSkip
}

// ResolveExtensions resolves each configured entry into osquery extension binaries.
// An entry may be a directory (scanned for files with the platform extension suffix),
// a specific extension binary file, or a glob pattern whose matches are resolved as
// directories or files. Symlinks are rejected everywhere (entries, glob matches, and
// directory contents) so the validated file is always the one osqueryd executes.
// It reports, per entry, the valid binaries and the ones skipped with a reason. It
// performs no logging so callers can use it both for autoload preparation and for
// diagnostics. osqueryd still applies its own safe-permission gate at load time
// (osquerybeat never passes --allow_unsafe), so unsafe binaries are additionally
// skipped by osqueryd and surfaced in diagnostics.
func ResolveExtensions(entries []string) []ExtensionResolveResult {
	if len(entries) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	results := make([]ExtensionResolveResult, 0, len(entries))
	for _, entry := range entries {
		res := ExtensionResolveResult{Entry: entry}
		switch {
		case entry == "":
			res.Error = "empty path"
		case !filepath.IsAbs(entry):
			res.Error = "path must be absolute"
		case isGlobPattern(entry):
			matches, err := filepath.Glob(entry)
			if err != nil {
				res.Error = fmt.Sprintf("invalid glob pattern: %v", err)
				break
			}
			sort.Strings(matches)
			for _, m := range matches {
				resolveExtensionPath(m, false, seen, &res)
			}
		default:
			resolveExtensionPath(entry, true, seen, &res)
		}
		results = append(results, res)
	}
	return results
}

// resolveExtensionPath resolves a single concrete path (a directory or a file) into
// extension binaries, appending results to res. When literal is true the path came
// directly from configuration (so a failure is an entry-level error); when false
// it came from a glob match (so failures are recorded as skips). Symlinks are
// rejected: the pre-check must validate the same file osqueryd will execute, and a
// link retargeted between check and load would bypass it.
func resolveExtensionPath(path string, literal bool, seen map[string]struct{}, res *ExtensionResolveResult) {
	fail := func(reason string) {
		if literal {
			res.Error = reason
		} else {
			res.Skipped = append(res.Skipped, ExtensionSkip{Path: path, Reason: reason})
		}
	}
	fi, err := os.Lstat(path)
	if err != nil {
		fail(err.Error())
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		fail("symlinks are not allowed")
		return
	}
	if fi.IsDir() {
		scanExtensionDir(path, seen, res)
		return
	}
	// A file selected explicitly (literal path or glob match) is autoloaded without a
	// suffix filter; only the safe-binary pre-check applies.
	addExtensionBinary(path, seen, res)
}

// scanExtensionDir adds every file with the platform extension suffix found directly
// in dir. os.ReadDir returns entries sorted by name, so the autoload content is
// deterministic. Symlinked candidates are recorded as skipped, not followed.
func scanExtensionDir(dir string, seen map[string]struct{}, res *ExtensionResolveResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		res.Skipped = append(res.Skipped, ExtensionSkip{Path: dir, Reason: err.Error()})
		return
	}
	suffix := extensionFileSuffix()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if e.Type()&os.ModeSymlink != 0 {
			res.Skipped = append(res.Skipped, ExtensionSkip{Path: path, Reason: "symlinks are not allowed"})
			continue
		}
		addExtensionBinary(path, seen, res)
	}
}

// addExtensionBinary validates a candidate binary path and appends it to the loaded
// or skipped list, deduplicating across all resolved entries.
func addExtensionBinary(path string, seen map[string]struct{}, res *ExtensionResolveResult) {
	if _, ok := seen[path]; ok {
		return
	}
	seen[path] = struct{}{}
	if err := ValidateExtensionPath(path); err != nil {
		res.Skipped = append(res.Skipped, ExtensionSkip{Path: path, Reason: err.Error()})
		return
	}
	res.Loaded = append(res.Loaded, path)
}

// isGlobPattern reports whether the entry contains glob metacharacters.
func isGlobPattern(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// extensionFileSuffix returns the file suffix osquery extension binaries use on
// the current platform.
func extensionFileSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ".ext"
}

// ValidateExtensionPath performs the beat-side pre-check for a customer-managed
// extension binary (absolute path, exists, regular file — not a symlink —,
// executable). Symlinks are rejected so the validated file is the one osqueryd
// executes. The ownership/writability safe-permission checks are enforced by
// osqueryd itself at load time (osquerybeat never passes --allow_unsafe).
func ValidateExtensionPath(p string) error {
	if p == "" {
		return errors.New("empty path")
	}
	if !filepath.IsAbs(p) {
		return errors.New("path must be absolute")
	}
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.New("symlinks are not allowed")
	}
	if !fi.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	// osquery requires an executable extension binary; the ownership/writability
	// (safe-permission) checks are enforced by osqueryd itself at load time.
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0 {
		return errors.New("file is not executable")
	}
	return nil
}

func verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath string) error {
	f, err := os.Open(extensionAutoloadPath)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		if i == 0 {
			// Check that the first line is the mandatory extension
			if line != mandatoryExtensionPath {
				return errors.New("extentsions autoload file is missing mandatory extension in the first line of the file")
			}
		}

		// Check that the line contains the valid path that exists
		_, err := os.Stat(line)
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (q *OSQueryD) prepareBinPath() error {
	// If path to osquery was not set use the current executable path
	if q.binPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		q.binPath = filepath.Dir(exePath)
	}
	return nil
}

func (q *OSQueryD) args(userFlags Flags) Args {
	flags := make(Flags, len(userFlags))

	// Copy user flags
	for k, userValue := range userFlags {
		// Skip enable_tables and disable_tables flags, they are set later in this function
		// after merging with default disabled tables
		if k == flagEnableTables || flagEnableTables == flagDisableTables {
			continue
		}
		flags[k] = userValue
	}

	// Copy protected flags, protected keys overwrite the user keys
	for k, v := range protectedFlags {
		flags[k] = v
	}

	flags["pidfile"] = q.resolveDataPath(flags.GetString("pidfile"))
	flags["database_path"] = q.resolveDataPath(flags.GetString("database_path"))
	flags["extensions_autoload"] = q.resolveDataPath(flags.GetString("extensions_autoload"))
	flags["flagfile"] = q.resolveDataPath(flags.GetString("flagfile"))

	flags["tls_server_certs"] = q.resolveCertsPath(flags.GetString("tls_server_certs"))

	// Augeas lenses are not available on windows
	if runtime.GOOS == "windows" {
		delete(flags, "augeas_lenses")
	} else {
		flags["augeas_lenses"] = q.lensesPath
	}

	flags["extensions_socket"] = q.socketPath

	if to := q.getExtensionsTimeout(); to > 0 {
		flags["extensions_timeout"] = to
	}

	// Extension names osqueryd must wait for at startup; queries do not run until
	// the required extensions have registered (or extensions_timeout elapses).
	if require := q.getExtensionRequire(); len(require) > 0 {
		flags["extensions_require"] = strings.Join(require, ",")
	}

	if q.configPlugin != "" {
		flags["config_plugin"] = q.configPlugin
	}

	if q.loggerPlugin != "" {
		flags["logger_plugin"] = q.loggerPlugin
	}

	if q.configRefreshInterval > 0 {
		flags["config_refresh"] = q.configRefreshInterval
	}

	// Set the appropriate logger_min_status flag based on osquerybeat log level
	// Map logp levels to osquery logger_min_status values: 1=WARNING, 2=ERROR
	var logMinStatus int
	level := zapcore.LevelOf(q.log.Core())
	switch {
	case level == zapcore.WarnLevel:
		logMinStatus = 1 // WARNING
	case level >= zapcore.ErrorLevel:
		logMinStatus = 2 // ERROR+
	}

	// osquery default is 0 (INFO/DEBUG) but we control that with the verbose flag already
	if logMinStatus > 0 {
		flags["logger_min_status"] = logMinStatus
		flags["disable_logging"] = false
	}

	if q.isVerbose() {
		flags["verbose"] = true
		flags["disable_logging"] = false
	}

	// Check enabled tables
	// If the default disabled table shows up in the enabled tables list, remove it from disabled tables list
	// This changes the behvaour for this flag in a sense that if `curl` table is enabled
	// then it just removes is from disabled tables flag and doesn't disable all the other table
	enabledTables, disabledTables := getEnabledDisabledTables(userFlags)
	if len(enabledTables) != 0 {
		flags[flagEnableTables] = strings.Join(enabledTables, ",")
	}

	if len(disabledTables) != 0 {
		flags[flagDisableTables] = strings.Join(disabledTables, ",")
	}

	return convertToArgs(flags)
}

func arrayToSet(arr []string) map[string]struct{} {
	m := make(map[string]struct{}, len(arr))
	for _, n := range arr {
		m[n] = struct{}{}
	}
	return m
}

// https://osquery.readthedocs.io/en/stable/installation/cli-flags/#enable-and-disable-flags
// By default every table is enabled.
// If a specific table is set in both --enable_tables and --disable_tables, disabling take precedence.
// If --enable_tables is defined and --disable_tables is not set, every table but the one defined in --enable_tables
func getEnabledDisabledTables(userFlags Flags) (enabled, disabled []string) {
	enabledTables := make(map[string]struct{})

	// Initialize with default disabled tables
	disabledTables := arrayToSet(defaultDisabledTables)

	iterate := func(key string, fn func(name string)) {
		if tablesValue, ok := userFlags[key]; ok {
			if tablesString, ok := tablesValue.(string); ok {
				tables := strings.Split(tablesString, ",")
				for _, table := range tables {
					name := strings.TrimSpace(table)
					if name == "" {
						continue
					}
					fn(name)
				}
			}
		}
	}

	normalize := func(tables map[string]struct{}) []string {
		res := make([]string, 0, len(tables))
		for name := range tables {
			res = append(res, name)
		}
		if len(res) > 0 {
			sort.Strings(res)
		}
		return res
	}

	// Append the disabled tables from flags
	iterate("disable_tables", func(name string) {
		disabledTables[name] = struct{}{}
	})

	// Check enabled tables flag and remove these tables from disabledTables
	iterate("enable_tables", func(name string) {
		if _, ok := disabledTables[name]; ok {
			delete(disabledTables, name)
		} else {
			enabledTables[name] = struct{}{}
		}
	})

	return normalize(enabledTables), normalize(disabledTables)
}

func (q *OSQueryD) createCommand(ctx context.Context, userFlags Flags) *exec.Cmd {
	//nolint:gosec // works as expected
	return exec.CommandContext(ctx,
		osquerydPath(q.binPath), q.args(userFlags)...)
}

func (q *OSQueryD) isVerbose() bool {
	return q.log.IsDebug()
}

func osquerydPath(dir string) string {
	return OsquerydPathForPlatform(runtime.GOOS, dir)
}

// OsquerydPathForPlatform returns the full path to osqueryd binary for platform
func OsquerydPathForPlatform(platform, dir string) string {
	if platform == "darwin" {
		return filepath.Join(dir, osqueryDarwinAppBundlePath, osquerydFilename(platform))

	}
	return filepath.Join(dir, osquerydFilename(platform))
}

func osquerydFilename(platform string) string {
	if platform == "windows" {
		return osqueryDName + ".exe"
	}
	return osqueryDName
}

func osqueryExtensionPath(dir string) string {
	return filepath.Join(dir, extensionName)
}

// OsqueryExtensionPathForPlatform returns the full path to osquery extension binary for platform.
func OsqueryExtensionPathForPlatform(platform, dir string) string {
	return filepath.Join(dir, osqueryExtensionFilename(platform))
}

func osqueryExtensionFilename(platform string) string {
	if platform == "windows" {
		return "osquery-extension.exe"
	}
	return "osquery-extension.ext"
}

func (q *OSQueryD) resolveDataPath(filename string) string {
	return filepath.Join(q.dataPath, filename)
}

func (q *OSQueryD) resolveCertsPath(filename string) string {
	return filepath.Join(q.certsPath, filename)
}

func (q *OSQueryD) logOSQueryOutput(ctx context.Context, r io.ReadCloser) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 2048), 64*1024) // 64KB max line size

	log := q.log.With("ctx", "osqueryd")

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try to parse structured osquery log format
		entry, err := osqlog.ParseGlogLine(line)
		if err != nil {
			// Failed to parse, log the raw line
			var level osqlog.Level
			if len(line) > 0 {
				level = osqlog.Level(line[0])
			} else {
				level = osqlog.LevelInfo
			}
			osqlog.LogWithLevel(log, level, line)
		} else {
			// Successfully parsed, log with structured fields
			entry.Log(log)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("error reading osqueryd output: %v", err)
		return err
	}

	return nil
}
