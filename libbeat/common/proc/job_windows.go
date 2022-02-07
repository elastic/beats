// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows
// +build windows

package proc

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Job is wrapper for windows JobObject
// https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects
// This helper guarantees a clean process tree kill on job handler close
type Job windows.Handle

var (
	// Public global JobObject should be initialized once in main
	JobObject Job
)

// CreateJobObject creates JobObject on Windows, global per process
// Should only be initialized once in main function
func CreateJobObject() (pj Job, err error) {
	if pj, err = NewJob(); err != nil {
		return
	}
	JobObject = pj
	return
}

// NewJob creates a instance of the JobObject
func NewJob() (Job, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {
		return 0, err
	}

	return Job(h), nil
}

// Close closes job handler
func (job Job) Close() error {
	if job == 0 {
		return nil
	}
	return windows.CloseHandle(windows.Handle(job))
}

// Assign assigns the process to the JobObject
func (job Job) Assign(p *os.Process) error {
	if job == 0 || p == nil {
		return nil
	}
	return windows.AssignProcessToJobObject(
		windows.Handle(job),
		windows.Handle((*process)(unsafe.Pointer(p)).Handle))
}

type process struct {
	Pid    int
	Handle uintptr
}
