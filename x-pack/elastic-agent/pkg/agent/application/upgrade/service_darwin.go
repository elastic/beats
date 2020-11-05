// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build darwin

package upgrade

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/hashicorp/go-multierror"
)

// Init initializes os dependent properties.
func (ch *CrashChecker) Init(ctx context.Context) error {
	ch.sc = &pidProvider{}

	return nil
}

type pidProvider struct{}

func (p *pidProvider) Close() {}

func (p *pidProvider) PID(ctx context.Context) (int, error) {
	piders := []func(context.Context) (int, error){
		// list of services differs when using sudo and not
		// agent should be included in sudo one but in case it's not
		// we're falling back to regular
		p.piderFromCmd(ctx, "sudo", "launchctl", "list", install.ServiceName),
		p.piderFromCmd(ctx, "launchctl", "list", install.ServiceName),
	}

	var pidErrors error
	for _, pider := range piders {
		pid, err := pider(ctx)
		if err == nil {
			return pid, nil
		}

		pidErrors = multierror.Append(pidErrors, err)
	}

	return 0, pidErrors
}

func (p *pidProvider) piderFromCmd(ctx context.Context, name string, args ...string) func(context.Context) (int, error) {
	return func(context.Context) (int, error) {
		listCmd := exec.Command(name, args...)
		listCmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: 0, Gid: 0},
		}
		out, err := listCmd.Output()
		if err != nil {
			return 0, errors.New("filed to read process id", err)
		}

		// find line
		pidLine := ""
		reader := bufio.NewReader(bytes.NewReader(out))
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, `"PID" = `) {
				pidLine = strings.TrimSpace(line)
				break
			}
		}

		if pidLine == "" {
			return 0, errors.New(fmt.Sprintf("service process not found for service '%v'", install.ServiceName))
		}

		re := regexp.MustCompile(`"PID" = ([0-9]+);`)
		matches := re.FindStringSubmatch(pidLine)
		if len(matches) != 2 {
			return 0, errors.New("could not detect pid of process", pidLine, matches)
		}

		pid, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, errors.New(fmt.Sprintf("filed to get process id[%v]", matches[1]), err)
		}

		return pid, nil
	}
}
