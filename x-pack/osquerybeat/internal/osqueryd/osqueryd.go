// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqueryd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
)

// The subdirectory to hold .pid, .db, .sock and other work file for osqueryd sub process. Open for discussion.
// Will see later what needs to be parameterized and what not.
const (
	osquerySubdir     = "osquery"
	extensionsTimeout = 10
)

type OsqueryD struct {
	RootDir    string
	SocketPath string
}

// TODO(AM): finalize what to do with config file, how much of the config file we need etc. Open question for now.
func (q *OsqueryD) Start(ctx context.Context) (<-chan error, error) {
	log := logp.NewLogger("osqueryd").With("dir", q.RootDir).With("socket_path", q.SocketPath)
	log.Info("Starting process")

	dir := filepath.Join(q.RootDir, osquerySubdir)

	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create dir %v, %w", dir, err)
	}

	cmd := q.createCommand(log, dir)

	cmd.SysProcAttr = setpgid()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var (
		errbuf strings.Builder
	)

	wait := func() error {
		if _, cerr := io.Copy(&errbuf, stderr); cerr != nil {
			return cerr
		}
		return cmd.Wait()
	}

	finished := make(chan error, 1)

	go func() {
		finished <- wait()
	}()

	done := make(chan error, 1)
	go func() {
		var ferr error
		select {
		case ferr = <-finished:
			if ferr != nil {
				s := strings.TrimSpace(errbuf.String())
				if s != "" {
					ferr = fmt.Errorf("%s: %w", s, ferr)
				}
			}
			if ferr != nil {
				log.Errorf("Process exited with error: %v", ferr)
			} else {
				log.Info("Process exited")
			}
		case <-ctx.Done():
			log.Info("Kill process group on context done")
			killProcessGroup(cmd)
			// Wait till finished
			<-finished
			ferr = ctx.Err()
		}
		done <- ferr
	}()

	return done, err
}

func (q *OsqueryD) createCommand(log *logp.Logger, dir string) *exec.Cmd {

	cmd := exec.Command(
		distro.OsquerydPath(q.RootDir),
		"--force=true",
		"--disable_watchdog",
		"--utc",
		"--pidfile="+path.Join(dir, "osquery.pid"),
		"--database_path="+path.Join(dir, "osquery.db"),
		"--extensions_socket="+q.SocketPath,
		"--config_path="+path.Join(dir, "osquery.conf"),
		"--logger_path="+dir,
		"--extensions_autoload="+path.Join(dir, "osquery.autoload"),
		fmt.Sprint("--extensions_timeout=", extensionsTimeout),
	)

	cmd.Args = append(cmd.Args, platformArgs()...)

	if log.IsDebug() {
		cmd.Args = append(cmd.Args, "--verbose")
	}
	return cmd
}
