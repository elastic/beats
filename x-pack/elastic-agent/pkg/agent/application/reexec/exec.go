// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package reexec

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func exec(exec string) error {
	args := []string{filepath.Base(exec)}
	args = append(args, os.Args[1:]...)
	return unix.Exec(exec, args, os.Environ())
}
