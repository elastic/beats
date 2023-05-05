// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Proc struct {
	Binary  string
	Args    []string
	Cmd     *exec.Cmd
	t       *testing.T
	tempDir string
}

// NewBeat createa a new Beat process from the system tests binary.
// It sets some requried options like the home path, logging, etc.
func NewBeat(t *testing.T, binary string, args []string, tempDir string) Proc {
	p := Proc{
		t:      t,
		Binary: binary,
		Args: append([]string{
			"--systemTest",
			// "-e",
			"--path.home", tempDir,
			"--path.logs", tempDir,
			"-E", "logging.to_files=true",
			"-E", "logging.files.rotateeverybytes=104857600", // About 100MB
		}, args...),
		tempDir: tempDir,
	}
	return p
}

func (p *Proc) Start() {
	fullPath, err := filepath.Abs(p.Binary)
	if err != nil {
		p.t.Fatalf("could got get full path from %q, err: %s", p.Binary, err)
	}
	p.Cmd = exec.Command(fullPath, p.Args...)

	if err := p.Cmd.Start(); err != nil {
		p.t.Fatalf("could not start process: %s", err)
	}
	p.t.Cleanup(func() {
		pid := p.Cmd.Process.Pid
		if err := p.Cmd.Process.Kill(); err != nil {
			p.t.Fatalf("could not stop process with PID: %d, err: %s", pid, err)
		}
	})
}

func (p *Proc) LogContains(s string, timeout time.Duration) bool {
	logFile := p.openLogFile()
	scanner := bufio.NewScanner(logFile)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		default:
			// fmt.Print(".")
			if scanner.Scan() {
				// fmt.Print("+")
				line := scanner.Text()
				// fmt.Println(line)
				if strings.Contains(line, s) {
					fmt.Println(line)
					return true
				}
			}
			// scanner.Scan() will return false when it reaches the end of the file,
			// then it will stop reading from the file.
			// So if it's error is nil, we create a new scanner
			if err := scanner.Err(); err == nil {
				scanner = bufio.NewScanner(logFile)
				// fmt.Println("got no error, creating new scanner")
			}
		case <-timer.C:
			p.t.Fatal("timeout")
		}
	}
}

// openLogFile opens the log file for reading and returns it.
// It also registers a cleanup function to close the file
// when the test ends.
func (p *Proc) openLogFile() *os.File {
	t := p.t
	glob := fmt.Sprintf("%s-*.ndjson", filepath.Join(p.tempDir, "filebeat"))
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not expand log file glob: %s", err)
	}
	t.Log("Glob:", glob, files)

	require.Eventually(t, func() bool {
		files, err = filepath.Glob(glob)
		if err != nil {
			t.Fatalf("could not expand log file glob: %s", err)
		}
		t.Log("Glob:", glob, files)
		if len(files) == 1 {
			return true
		}

		return false
	}, 5*time.Second, 100*time.Millisecond, "waiting for log file")
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
	t.Cleanup(func() { f.Close() })
	return f
}
