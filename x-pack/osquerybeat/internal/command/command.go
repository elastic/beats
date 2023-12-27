// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func Execute(ctx context.Context, name string, arg ...string) (out string, err error) {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	var (
		outbuf strings.Builder
		errbuf strings.Builder
	)

	finished := make(chan error, 1)

	wait := func() error {
		_, err := io.Copy(&outbuf, stdout)
		if err != nil {
			return err
		}

		_, err = io.Copy(&errbuf, stderr)
		if err != nil {
			return err
		}
		return cmd.Wait()
	}

	go func() {
		finished <- wait()
	}()

	// Wait either on process finish or context cancel
	select {
	case err = <-finished:
		if err != nil {
			s := strings.TrimSpace(errbuf.String())
			if s == "" {
				return
			}
			return "", fmt.Errorf("%s: %w", s, err)
		}
	case <-ctx.Done():
		_ = cmd.Process.Kill()
	}

	return outbuf.String(), nil
}
