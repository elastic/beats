// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package proc

import (
	"os"
)

// Job is noop on unix
type Job int

var (
	// Public global JobObject, 0 value on linux
	JobObject Job
)

func CreateJobObject() (pj Job, err error) {
	return
}

// NewJob is noop on unix
func NewJob() (Job, error) {
	return 0, nil
}

// Close is noop on unix
func (job Job) Close() error {
	return nil
}

// Assign is noop on unix
func (job Job) Assign(p *os.Process) error {
	return nil
}
