// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !darwin
// +build !windows

package upgrade

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
)

const (
	// delay after agent restart is performed to allow agent to tear down all the processes
	// important mainly for windows, as it prevents removing files which are in use
	afterRestartDelay = 2 * time.Second
)

// Init initializes os dependent properties.
func (ch *CrashChecker) Init(ctx context.Context) error {
	dbusConn, err := dbus.New()
	if err != nil {
		return errors.New("failed to create dbus connection", err)
	}

	ch.sc = &pidProvider{
		dbusConn: dbusConn,
	}

	return nil
}

type pidProvider struct {
	dbusConn *dbus.Conn
}

func (p *pidProvider) Close() {
	p.dbusConn.Close()
}

func (p *pidProvider) PID(ctx context.Context) (int, error) {
	prop, err := p.dbusConn.GetServiceProperty(install.ServiceName, "MainPID")
	if err != nil {
		return 0, errors.New("failed to read service", err)
	}

	pid, ok := prop.Value.Value().(uint32)
	if !ok {
		return 0, errors.New("failed to get process id", prop.Value.Value())
	}

	return int(pid), nil
}

func invokeCmd() *exec.Cmd {
	homeExePath := filepath.Join(paths.Home(), agentName)

	cmd := exec.Command(homeExePath, watcherSubcommand,
		"--path.config", paths.Config(),
		"--path.home", paths.Top(),
	)

	var cred = &syscall.Credential{
		Uid:         uint32(os.Getuid()),
		Gid:         uint32(os.Getgid()),
		Groups:      nil,
		NoSetGroups: true,
	}
	var sysproc = &syscall.SysProcAttr{
		Credential: cred,
		Setsid:     true,
		// propagate sigint instead of sigkill so we can ignore it
		Pdeathsig: syscall.SIGINT,
	}
	cmd.SysProcAttr = sysproc
	return cmd
}
