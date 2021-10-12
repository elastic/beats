// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func createListener(log *logger.Logger) (net.Listener, error) {
	path := strings.TrimPrefix(control.Address(), "unix://")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		cleanupListener(log)
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}
	lis, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	err = os.Chmod(path, 0700)
	if err != nil {
		// failed to set permissions (close listener)
		lis.Close()
		return nil, err
	}
	return lis, err
}

func cleanupListener(log *logger.Logger) {
	path := strings.TrimPrefix(control.Address(), "unix://")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Debug("%s", errors.New(err, fmt.Sprintf("Failed to cleanup %s", path), errors.TypeFilesystem, errors.M("path", path)))
	}
}
