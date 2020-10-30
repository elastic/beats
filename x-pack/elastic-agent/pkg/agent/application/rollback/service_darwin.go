// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build darwin

package rollback

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
)

// Init initializes os dependent properties.
func (ch *CrashChecker) Init(ctx context.Context) error {
	ch.sc = &pidProvider{}

	return nil
}

type pidProvider struct{}

func (p *pidProvider) Close() {}

func (p *pidProvider) PID(ctx context.Context) (int, error) {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return 0, errors.New("filed to read process id", err)
	}

	// find line
	serviceLine := ""
	reader := bufio.NewReader(bytes.NewReader(out))
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, install.ServiceDisplayName) {
			serviceLine = strings.TrimSpace(line)
			break
		}
	}

	if serviceLine == "" {
		return 0, errors.New(fmt.Sprintf("service process not found for service '%v'", install.ServiceDisplayName))
	}

	re := regexp.MustCompile(`"PID" = ([0-9]+);`)
	matches := re.FindStringSubmatch(serviceLine)
	if len(matches) != 2 {
		return 0, errors.New("could not detect pid of process")
	}

	pid, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, errors.New(fmt.Sprintf("filed to get process id[%v]", matches[1]), err)
	}

	return pid, nil
}
