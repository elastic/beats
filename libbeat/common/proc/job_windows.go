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
	"fmt"
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
		return pj, err
	}
	JobObject = pj
	return pj, nil
}

// NewJob creates a instance of the JobObject
func NewJob() (Job, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	// From https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects
	// ... if the job has the JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE flag specified,
	// closing the last job object handle terminates all associated processes
	// and then destroys the job object itself.
	// If a nested job has the JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE flag specified,
	// closing the last job object handle terminates all processes associated
	// with the job and its child jobs in the hierarchy.
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

	// To assign a process to a job, you need a handle to the process. Since os.Process provides no
	// way to obtain it's underlying handle safely, get one with OpenProcess().
	//   https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-openprocess
	// This requires at least the PROCESS_SET_QUOTA and PROCESS_TERMINATE access rights.
	//   https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-assignprocesstojobobject
	desiredAccess := uint32(windows.PROCESS_SET_QUOTA | windows.PROCESS_TERMINATE)
	processHandle, err := windows.OpenProcess(desiredAccess, false, uint32(p.Pid)) //nolint:gosec // G115 Conversion from int to uint32 is safe here.
	if err != nil {
		return fmt.Errorf("opening process handle: %w", err)
	}
	defer windows.CloseHandle(processHandle) //nolint:errcheck // No way to handle errors returned here so safe to ignore.

	err = windows.AssignProcessToJobObject(windows.Handle(job), processHandle)
	if err != nil {
		return fmt.Errorf("assigning to job object: %w", err)
	}

	return nil
}

type process struct {
	Pid    int
	Handle uintptr
}
