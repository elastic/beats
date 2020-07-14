// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !linux
// +build !darwin

package process

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func getCmd(logger *logger.Logger, path string, env []string, uid, gid int, arg ...string) *exec.Cmd {
	cmd := exec.Command(path, arg...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Dir = filepath.Dir(path)

	return cmd
}
