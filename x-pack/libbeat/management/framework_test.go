// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type Proc struct {
	Binary string
	Args   []string
	Cmd    *exec.Cmd
	t      *testing.T
	stdout *bufio.Scanner
}

func NewProc(t *testing.T, binary string, args []string, port int) Proc {
	p := Proc{
		t:      t,
		Binary: binary,
		Args: append([]string{
			"--systemTest",
			"-e",
			"-d",
			// "*",
			"centralmgmt, centralmgmt.V2-manager",
			"-E",
			fmt.Sprintf("management.insecure_grpc_url_for_testing=\"localhost:%d\"", port),
			"-E",
			"management.enabled=true",
		}, args...),
	}
	return p
}

func (p *Proc) Start() {
	fullPath, err := filepath.Abs(p.Binary)
	if err != nil {
		p.t.Fatalf("could got get full path from %q, err: %s", p.Binary, err)
	}
	p.Cmd = exec.Command(fullPath, p.Args...)
	stdout, err := p.Cmd.StderrPipe()
	if err != nil {
		p.t.Fatalf("could not get stdout pipe for process, err: %s", err)
	}
	p.stdout = bufio.NewScanner(stdout)

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
	p.t.Log("LogContans called")
	defer p.t.Log("LogContans done")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		default:
			if p.stdout.Scan() {
				line := p.stdout.Text()
				if strings.Contains(line, s) {
					fmt.Println(line)
					return true
				}
			}
		case <-timer.C:
			p.t.Fatal("timeout")
		}
	}
}
