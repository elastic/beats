// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package operations

import "os"

// RunningUnderSupervisor returns true when executing Agent is running under
// the supervisor processes of the OS.
func RunningUnderSupervisor() bool {
	return os.Getppid() == 1
}
