// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !darwin && !windows
// +build !darwin,!windows

package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	// delay after agent restart is performed to allow agent to tear down all the processes
	// important mainly for windows, as it prevents removing files which are in use
	afterRestartDelay = 2 * time.Second
)

type pidProvider interface {
	Init() error
	Close()
	PID(ctx context.Context) (int, error)
	Name() string
}

// Init initializes os dependent properties.
func (ch *CrashChecker) Init(ctx context.Context, _ *logger.Logger) error {
	pp := relevantPidProvider()
	pp.Init()

	ch.sc = pp

	return nil
}

func relevantPidProvider() pidProvider {
	var pp pidProvider

	switch {
	case isSystemd():
		pp = &dbusPidProvider{}
	case isUpstart():
		pp = &upstartPidProvider{}
	case isSysv():
		pp = &sysvPidProvider{}
	default:
		// in case we're using unsupported service manager
		// let other checks work
		pp = &noopPidProvider{}
	}

	return pp
}

// Upstart PID Provider

type upstartPidProvider struct{}

func (p *upstartPidProvider) Init() error { return nil }

func (p *upstartPidProvider) Close() {}

func (p *upstartPidProvider) Name() string { return "Upstart" }

func (p *upstartPidProvider) PID(ctx context.Context) (int, error) {
	listCmd := exec.Command("/sbin/status", agentName)
	out, err := listCmd.Output()
	if err != nil {
		return 0, errors.New("failed to read process id", err)
	}

	// find line
	pidLine := strings.TrimSpace(string(out))
	if pidLine == "" {
		return 0, errors.New(fmt.Sprintf("service process not found for service '%v'", paths.ServiceName))
	}

	re := regexp.MustCompile(agentName + ` start/running, process ([0-9]+)`)
	matches := re.FindStringSubmatch(pidLine)
	if len(matches) != 2 {
		return 0, errors.New("could not detect pid of process", pidLine, matches)
	}

	pid, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to get process id[%v]", matches[1]), err)
	}

	return pid, nil
}

// SYSV PID Provider

type sysvPidProvider struct{}

func (p *sysvPidProvider) Init() error { return nil }

func (p *sysvPidProvider) Close() {}

func (p *sysvPidProvider) Name() string { return "SysV" }

func (p *sysvPidProvider) PID(ctx context.Context) (int, error) {
	listCmd := exec.Command("service", agentName, "status")
	out, err := listCmd.Output()
	if err != nil {
		return 0, errors.New("failed to read process id", err)
	}

	// find line
	statusLine := strings.TrimSpace(string(out))
	if statusLine == "" {
		return 0, errors.New(fmt.Sprintf("service process not found for service '%v'", paths.ServiceName))
	}

	// sysv does not report pid, let's do best effort
	if !strings.HasPrefix(statusLine, "Running") {
		return 0, errors.New(fmt.Sprintf("'%v' is not running", paths.ServiceName))
	}

	pidofLine, err := exec.Command("pidof", filepath.Join(paths.InstallPath, paths.BinaryName)).Output()
	if err != nil {
		return 0, errors.New(fmt.Sprintf("PID not found for'%v': %v", paths.ServiceName, err))
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidofLine)))
	if err != nil {
		return 0, errors.New("PID not a number")
	}

	return pid, nil
}

// DBUS PID provider

type dbusPidProvider struct {
	dbusConn *dbus.Conn
}

func (p *dbusPidProvider) Init() error {
	dbusConn, err := dbus.New()
	if err != nil {
		return errors.New("failed to create dbus connection", err)
	}

	p.dbusConn = dbusConn
	return nil
}

func (p *dbusPidProvider) Name() string { return "DBus" }

func (p *dbusPidProvider) Close() {
	p.dbusConn.Close()
}

func (p *dbusPidProvider) PID(ctx context.Context) (int, error) {
	sn := paths.ServiceName
	if !strings.HasSuffix(sn, ".service") {
		sn += ".service"
	}

	prop, err := p.dbusConn.GetServiceProperty(sn, "MainPID")
	if err != nil {
		return 0, errors.New("failed to read service", err)
	}

	pid, ok := prop.Value.Value().(uint32)
	if !ok {
		return 0, errors.New("failed to get process id", prop.Value.Value())
	}

	return int(pid), nil
}

// noop PID provider

type noopPidProvider struct{}

func (p *noopPidProvider) Init() error { return nil }

func (p *noopPidProvider) Close() {}

func (p *noopPidProvider) Name() string { return "noop" }

func (p *noopPidProvider) PID(ctx context.Context) (int, error) { return 0, nil }

func invokeCmd(topPath string) *exec.Cmd {
	homeExePath := filepath.Join(topPath, agentName)

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

func isSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}

	if _, err := os.Stat("/proc/1/comm"); err == nil {
		filerc, err := os.Open("/proc/1/comm")
		if err != nil {
			return false
		}
		defer filerc.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(filerc)
		contents := buf.String()

		if strings.Trim(contents, " \r\n") == "systemd" {
			return true
		}
	}
	return false
}

func isUpstart() bool {
	if _, err := os.Stat("/sbin/upstart-udev-bridge"); err == nil {
		return true
	}

	if _, err := os.Stat("/sbin/initctl"); err == nil {
		if out, err := exec.Command("/sbin/initctl", "--version").Output(); err == nil {
			if bytes.Contains(out, []byte("initctl (upstart")) {
				return true
			}
		}
	}
	return false
}

func isSysv() bool {
	// PID 1 is init
	out, err := exec.Command("sudo", "cat", "/proc/1/comm").Output()
	if err != nil {
		o, err := exec.Command("cat", "/proc/1/comm").Output()
		if err != nil {
			return false
		}
		out = o
	}

	if strings.TrimSpace(string(out)) != "init" {
		return false
	}

	// /sbin/init is not a link
	initFile, err := os.Open("/sbin/init")
	if err != nil || initFile == nil {
		return false
	}

	fi, err := initFile.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != os.ModeSymlink
}
