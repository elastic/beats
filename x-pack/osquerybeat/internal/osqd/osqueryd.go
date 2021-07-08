// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dolmen-go/contextio"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	osqueryDName    = "osqueryd"
	osqueryAutoload = "osquery.autoload"
)

const (
	defaultExtensionsTimeout     = 10
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
func (q *OSQueryD) Run(ctx context.Context) error {
	cleanup, err := q.prepare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := q.createCommand()

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
		select {
		case finished <- wait():
		}
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
			return nil, errors.Wrapf(err, "could not stat extension path")
		}
	}

	// Write the autoload file
	extensionAutoloadPath := q.osqueryAutoloadPath()
	if err := ioutil.WriteFile(extensionAutoloadPath, []byte(extensionPath), 0644); err != nil {
		return nil, errors.Wrap(err, "could not write osquery extension autoload file")
	}

	return func() {}, nil
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

func (q *OSQueryD) createCommand() *exec.Cmd {

	cmd := exec.Command(
		osquerydPath(q.binPath),
		"--force=true",
		"--disable_watchdog",
		"--utc",
		// // Enable events collection
		// "--disable_events=false",
		// // Begin: enable process events audit
		// "--disable_audit=false",
		// "--audit_allow_config=true",
		// "--audit_persist=true",
		// "--audit_allow_process_events=true",
		// // End: enable process events audit

		// // Begin: enable sockets audit
		// "--audit_allow_sockets=true",
		// "--audit_allow_unix=true", // Allow domain sockets audit
		// // End: enable sockets audit

		// // Setting this value to 1 will auto-clear events whenever a SELECT is performed against the table, reducing all impact of the buffer.
		// "--events_expiry=1",

		"--pidfile="+path.Join(q.dataPath, "osquery.pid"),
		"--database_path="+path.Join(q.dataPath, "osquery.db"),
		"--extensions_socket="+q.socketPath,
		"--logger_path="+q.dataPath,
		"--extensions_autoload="+q.osqueryAutoloadPath(),
		"--extensions_interval=3",
		fmt.Sprint("--extensions_timeout=", q.extensionsTimeout),
	)

	if q.configPlugin != "" {
		cmd.Args = append(cmd.Args, "--config_plugin="+q.configPlugin)
	}

	if q.loggerPlugin != "" {
		cmd.Args = append(cmd.Args, "--logger_plugin="+q.loggerPlugin)
	}

	if q.configRefreshInterval > 0 {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--config_refresh=%d", q.configRefreshInterval))
	}

	cmd.Args = append(cmd.Args, platformArgs()...)

	if q.isVerbose() {
		cmd.Args = append(cmd.Args, "--verbose")
		cmd.Args = append(cmd.Args, "--disable_logging=false")
	}
	return cmd
}

func (q *OSQueryD) isVerbose() bool {
	return q.log.IsDebug()
}

func osquerydPath(dir string) string {
	return filepath.Join(dir, osquerydFilename())
}

func osqueryExtensionPath(dir string) string {
	return filepath.Join(dir, extensionName)
}

func (q *OSQueryD) osqueryAutoloadPath() string {
	return filepath.Join(q.dataPath, osqueryAutoload)
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
