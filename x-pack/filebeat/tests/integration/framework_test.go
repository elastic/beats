// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type BeatProc struct {
	Binary  string
	Args    []string
	Cmd     *exec.Cmd
	t       *testing.T
	tempDir string
}

// NewBeat createa a new Beat process from the system tests binary.
// It sets some requried options like the home path, logging, etc.
func NewBeat(t *testing.T, binary string, args []string, tempDir string) BeatProc {
	p := BeatProc{
		t:      t,
		Binary: binary,
		Args: append([]string{
			"--systemTest",
			"--path.home", tempDir,
			"--path.logs", tempDir,
			"-E", "logging.to_files=true",
			"-E", "logging.files.rotateeverybytes=104857600", // About 100MB
		}, args...),
		tempDir: tempDir,
	}
	return p
}

func (b *BeatProc) Start() {
	fullPath, err := filepath.Abs(b.Binary)
	if err != nil {
		b.t.Fatalf("could got get full path from %q, err: %s", b.Binary, err)
	}
	b.Cmd = exec.Command(fullPath, b.Args...)

	if err := b.Cmd.Start(); err != nil {
		b.t.Fatalf("could not start process: %s", err)
	}
	b.t.Cleanup(func() {
		pid := b.Cmd.Process.Pid
		if err := b.Cmd.Process.Kill(); err != nil {
			b.t.Fatalf("could not stop process with PID: %d, err: %s", pid, err)
		}
	})
}

// LogContains looks for s as a sub string of every log line,
// it will open the log file on every call, read it until EOF,
// then close it.
func (b *BeatProc) LogContains(s string) bool {
	logFile := b.openLogFile()
	defer func() {
		if err := logFile.Close(); err != nil {
			// That's not quite a test error, but it can impact
			// next executions of LogContains, so treat it as an error
			b.t.Errorf("could not close log file: %s", err)
		}
	}()
	scanner := bufio.NewScanner(logFile)

	// TODO(Tiago) Remove this very verbose debugging code
	startTime := time.Now()
	linesScanned := 0
	defer func() {
		b.t.Logf("lines scanned: %d", linesScanned)
		pos, err := logFile.Seek(0, io.SeekCurrent)
		if err != nil {
			b.t.Errorf("could not seek file '%s': %s", logFile.Name(), err)
		}
		b.t.Logf("last position on '%s': %d", logFile.Name(), pos)
		b.t.Logf("took %s", time.Now().Sub(startTime).String())
	}()
	for scanner.Scan() {
		linesScanned++
		if strings.Contains(scanner.Text(), s) {
			return true
		}
	}

	fstat, err := logFile.Stat()
	if err != nil {
		b.t.Logf("cannot stat file: %s:", err)
	}
	b.t.Logf("[Stat] Name: %s, Size %d, ModTime: %s, Sys: %#v", fstat.Name(), fstat.Size(), fstat.ModTime().Format(time.RFC3339), fstat.Sys())

	return false
}

// openLogFile opens the log file for reading and returns it.
// It also registers a cleanup function to close the file
// when the test ends.
func (b *BeatProc) openLogFile() *os.File {
	t := b.t
	glob := fmt.Sprintf("%s-*.ndjson", filepath.Join(b.tempDir, "filebeat"))
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not expand log file glob: %s", err)
	}

	require.Eventually(t, func() bool {
		files, err = filepath.Glob(glob)
		if err != nil {
			t.Fatalf("could not expand log file glob: %s", err)
		}
		if len(files) == 1 {
			return true
		}

		return false
	}, 5*time.Second, 100*time.Millisecond,
		"waiting for log file matching glob '%s' to be created", glob)

	// On a normal operation there must be a single log, if there are more
	// than one, then there is an issue and the Beat is logging too much,
	// which is enough to stop the test
	if len(files) != 1 {
		t.Fatalf("there must be only one log file for %s, found: %d",
			glob, len(files))
	}

	f, err := os.Open(files[0])
	if err != nil {
		t.Fatalf("could not open log file '%s': %s", files[0], err)
	}

	t.Logf("file: '%s' successfully opened", files[0])
	return f
}
