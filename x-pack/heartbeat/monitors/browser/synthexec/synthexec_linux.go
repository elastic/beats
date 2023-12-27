// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package synthexec

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/heartbeat/security"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	platformCmdMutate = func(cmd *exec.Cmd) {
		logp.L().Warn("invoking node as:", security.NodeChildProcCred, " from: ", os.Getenv("HOME"))
		// Note that while cmd.SysProcAtr takes a syscall.SysProcAttr object
		// we are passing in a unix.SysProcAttr object
		// this is equivalent, but the unix package is not considered deprecated
		// as the syscall package is
		cmd.SysProcAttr = &unix.SysProcAttr{
			// Ensure node subprocesses are killed if this process dies (linux only)
			Pdeathsig: unix.SIGKILL,
			// Apply restricted user if available
			Credential: security.NodeChildProcCred,
		}
	}
}
