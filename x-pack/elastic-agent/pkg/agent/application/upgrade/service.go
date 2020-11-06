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

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
)

// Init initializes os dependent properties.
func (ch *CrashChecker) Init(ctx context.Context) error {
	// TODO: use with context once dbus upgraded
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
		return 0, errors.New("filed to read service", err)
	}

	pid, ok := prop.Value.Value().(uint32)
	if !ok {
		return 0, errors.New("filed to get process id", prop.Value.Value())
	}

	return int(pid), nil
}

func getInvokeCmd() *exec.Cmd {
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
