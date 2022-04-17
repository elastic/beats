// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dolmen-go/contextio"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/libbeat/common/proc"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/fileutil"
)

const (
	osqueryDName               = "osqueryd"
	osqueryDarwinAppBundlePath = "osquery.app/Contents/MacOS"
)

const (
	defaultExitTimeout           = 10 * time.Second
	defaultDataDir               = "osquery"
	defaultConfigRefreshInterval = 30 // interval osqueryd will poll for configuration changed; scheduled queries configuration for now
)

type OSQueryD struct {
	socketPath string
	binPath    string
	dataPath   string

	configPlugin string
	loggerPlugin string

	extensionsTimeout     int
	configRefreshInterval int

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

func New(socketPath string, opts ...Option) *OSQueryD {
	q := &OSQueryD{
		socketPath:            socketPath,
		extensionsTimeout:     defaultExtensionsTimeout,
		configRefreshInterval: defaultConfigRefreshInterval,
	}

	for _, opt := range opts {
		opt(q)
	}

	if q.dataPath == "" {
		q.dataPath = filepath.Join(q.binPath, defaultDataDir)
	}

	return q
}

func (q *OSQueryD) SocketPath() string {
	return q.socketPath
}

func (q *OSQueryD) DataPath() string {
	return q.dataPath
}

// Check checks if the binary exists and executable
func (q *OSQueryD) Check(ctx context.Context) error {
	err := q.prepareBinPath()
	if err != nil {
		return fmt.Errorf("failed to prepare bin path, %w", err)
	}

	cmd := exec.Command(
		osquerydPath(q.binPath),
		"--S",
		"--version",
	)

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()

	return nil
}

// Run executes osqueryd binary as a child process
func (q *OSQueryD) Run(ctx context.Context, flags Flags) error {
	cleanup, err := q.prepare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := q.createCommand(flags)

	q.log.Debugf("start osqueryd process: args: %v", cmd.Args)

	cmd.SysProcAttr = setpgid()

	// Read standard output
	var wg sync.WaitGroup

	if q.isVerbose() {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.logOSQueryOutput(ctx, stdout)
		}()
	}

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

	var (
		errbuf strings.Builder
	)

	ctxstderr := contextio.NewReader(ctx, stderr)
	wait := func() error {
		if _, cerr := io.Copy(&errbuf, ctxstderr); cerr != nil {
			return cerr
		}
		return cmd.Wait()
	}

	finished := make(chan error, 1)

	// Wait on osqueryd exit
	wg.Add(1)
	go func() {
		defer wg.Done()
		finished <- wait()
	}()

	select {
	case err = <-finished:
		if err != nil {
			s := strings.TrimSpace(errbuf.String())
			if s != "" {
				err = fmt.Errorf("%s: %w", s, err)
			}
		}
		if err != nil {
			q.log.Errorf("process exited with error: %v", err)
		} else {
			q.log.Info("process exited")
		}
	case <-ctx.Done():
		q.log.Debug("kill process group on context done")
		killProcessGroup(cmd)
		// Wait till finished
		<-finished
	}

	wg.Wait()

	return err
}

func (q *OSQueryD) prepare(ctx context.Context) (func(), error) {
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
	extensionPath := osqueryExtensionPath(q.binPath)
	if _, err := os.Stat(extensionPath); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "extension path does not exist: %s", extensionPath)
		} else {
			return nil, errors.Wrapf(err, "failed to stat extension path")
		}
	}

	// Write the autoload file
	extensionAutoloadPath := q.resolveDataPath(osqueryAutoload)
	err = prepareAutoloadFile(extensionAutoloadPath, extensionPath, q.log)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare extensions autoload file")
	}

	// Write the flagsfile in order to lock down/prevent loading default flags from osquery global locations.
	// Otherwise the osqueryi and osqueryd will try to load the default flags file,
	// for example from /var/osquery/osquery.flags.default on Mac, and can potentially mess up configuration of our osquery instance.
	flagsfilePath := q.resolveDataPath(osqueryFlagfile)
	exists, err := fileutil.FileExists(flagsfilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check flagsfile path")
	}
	if !exists {
		f, err := os.OpenFile(flagsfilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create flagsfile")
		}
		f.Close()
	}

	return func() {}, nil
}

func prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath string, log *logp.Logger) error {
	ok, err := fileutil.FileExists(extensionAutoloadPath)
	if err != nil {
		return errors.Wrapf(err, "failed to check osquery.autoload file exists")
	}

	rewrite := false

	if ok {
		log.Debugf("Extensions autoload file %s exists, verify the first extension is ours", extensionAutoloadPath)
		err = verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath)
		if err != nil {
			log.Debugf("Extensions autoload file %v verification failed, err: %v, create a new one", extensionAutoloadPath, err)
			rewrite = true
		}
	} else {
		log.Debugf("Extensions autoload file %s doesn't exists, create a new one", extensionAutoloadPath)
		rewrite = true
	}

	if rewrite {
		if err := ioutil.WriteFile(extensionAutoloadPath, []byte(mandatoryExtensionPath), 0644); err != nil {
			return errors.Wrap(err, "failed write osquery extension autoload file")
		}
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

	flags["extensions_socket"] = q.socketPath

	if q.extensionsTimeout > 0 {
		flags["extensions_timeout"] = q.extensionsTimeout

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

	if q.isVerbose() {
		flags["verbose"] = true
		flags["disable_logging"] = false
	}

	return convertToArgs(flags)
}

func (q *OSQueryD) createCommand(userFlags Flags) *exec.Cmd {
	return exec.Command(
		osquerydPath(q.binPath), q.args(userFlags)...)
}

func (q *OSQueryD) isVerbose() bool {
	return q.log.IsDebug()
}

func osquerydPath(dir string) string {
	return QsquerydPathForPlatform(runtime.GOOS, dir)
}

// QsquerydPathForPlatform returns the full path to osqueryd binary for platform
func QsquerydPathForPlatform(platform, dir string) string {
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

func (q *OSQueryD) resolveDataPath(filename string) string {
	return filepath.Join(q.dataPath, filename)
}

func (q *OSQueryD) logOSQueryOutput(ctx context.Context, r io.ReadCloser) error {
	log := q.log.With("ctx", "osqueryd output")

	buf := make([]byte, 2048, 2048)
LOOP:
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			log.Info(string(buf[:n]))
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			break LOOP
		default:
		}
	}
	return nil
}
