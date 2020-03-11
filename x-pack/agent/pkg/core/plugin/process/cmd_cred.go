// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin

package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

func getCmd(logger *logger.Logger, path string, env []string, uid, gid int, arg ...string) *exec.Cmd {
	cmd := exec.Command(path, arg...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Dir = filepath.Dir(path)
	if isInt32(uid) && isInt32(gid) {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid:         uint32(uid),
				Gid:         uint32(gid),
				NoSetGroups: true,
			},
		}
	} else {
		logger.Errorf("provided uid or gid for %s is invalid. uid: '%d' gid: '%d'.", path, uid, gid)
	}

	return cmd
}
