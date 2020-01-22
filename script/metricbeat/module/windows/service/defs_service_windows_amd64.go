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

// Created by cgo -godefs - DO NOT EDIT
// cgo.exe -godefs defs_service_windows.go

package service

type ServiceErrno uintptr

const (
	SERVICE_ERROR_ACCESS_DENIED           ServiceErrno = 0x5
	SERVICE_ERROR_MORE_DATA               ServiceErrno = 0xea
	SERVICE_ERROR_INVALID_PARAMETER       ServiceErrno = 0x57
	SERVICE_ERROR_INVALID_HANDLE          ServiceErrno = 0x6
	SERVICE_ERROR_INVALID_LEVEL           ServiceErrno = 0x7c
	SERVICE_ERROR_INVALID_NAME            ServiceErrno = 0x7b
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS    ServiceErrno = 0x45b
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST ServiceErrno = 0x429
	SERVICE_ERROR_INSUFFICIENT_BUFFER     ServiceErrno = 0x7a
	SERVICE_ERROR_SERVICE_DOES_NOT_EXIST  ServiceErrno = 0x424
)

type ServiceErrorControl uint32

const (
	SERVICE_ERROR_CRITICAL ServiceErrno = 0x3
	SERVICE_ERROR_IGNORE   ServiceErrno = 0x0
	SERVICE_ERROR_NORMAL   ServiceErrno = 0x1
	SERVICE_ERROR_SEVERE   ServiceErrno = 0x2
)

var serviceErrors = map[ServiceErrno]struct{}{
	SERVICE_ERROR_ACCESS_DENIED:           struct{}{},
	SERVICE_ERROR_MORE_DATA:               struct{}{},
	SERVICE_ERROR_INVALID_PARAMETER:       struct{}{},
	SERVICE_ERROR_INVALID_HANDLE:          struct{}{},
	SERVICE_ERROR_INVALID_LEVEL:           struct{}{},
	SERVICE_ERROR_INVALID_NAME:            struct{}{},
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS:    struct{}{},
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST: struct{}{},
	SERVICE_ERROR_INSUFFICIENT_BUFFER:     struct{}{},
	SERVICE_ERROR_CRITICAL:                struct{}{},
	SERVICE_ERROR_IGNORE:                  struct{}{},
	SERVICE_ERROR_NORMAL:                  struct{}{},
	SERVICE_ERROR_SEVERE:                  struct{}{},
	SERVICE_ERROR_SERVICE_DOES_NOT_EXIST:  struct{}{},
}

type ServiceType uint32

const (
	ServiceDriver ServiceType = 0xb

	ServiceFileSystemDriver ServiceType = 0x2

	ServiceKernelDriver ServiceType = 0x1

	ServiceWin32 ServiceType = 0x30

	ServiceWin32OwnProcess ServiceType = 0x10

	ServiceWin32Shareprocess  ServiceType = 0x20
	ServiceInteractiveProcess ServiceType = 0x100
)

type ServiceState uint32

const (
	ServiceContinuePending ServiceState = 0x5
	ServicePausePending    ServiceState = 0x6
	ServicePaused          ServiceState = 0x7
	ServiceRunning         ServiceState = 0x4
	ServiceStartPending    ServiceState = 0x2
	ServiceStopPending     ServiceState = 0x3
	ServiceStopped         ServiceState = 0x1
)

type ServiceEnumState uint32

const (
	ServiceActive ServiceEnumState = 0x1

	ServiceInActive ServiceEnumState = 0x2

	ServiceStateAll ServiceEnumState = 0x3
)

type ServiceSCMAccessRight uint32

const (
	ScManagerAllAccess ServiceSCMAccessRight = 0xf003f

	ScManagerConnect ServiceSCMAccessRight = 0x1

	ScManagerEnumerateService ServiceSCMAccessRight = 0x4

	ScManagerQueryLockStatus ServiceSCMAccessRight = 0x10
)

type ServiceAccessRight uint32

const (
	ServiceAllAccess ServiceAccessRight = 0xf01ff

	ServiceChangeConfig ServiceAccessRight = 0x2

	ServiceEnumerateDependents ServiceAccessRight = 0x8

	ServiceInterrogate ServiceAccessRight = 0x80

	ServicePauseContinue ServiceAccessRight = 0x40

	ServiceQueryConfig ServiceAccessRight = 0x1

	ServiceQueryStatus ServiceAccessRight = 0x4

	ServiceStart ServiceAccessRight = 0x10

	ServiceStop ServiceAccessRight = 0x20

	ServiceUserDefinedControl ServiceAccessRight = 0x100
)

type ServiceInfoLevel uint32

const (
	ScEnumProcessInfo ServiceInfoLevel = 0x0
)

type ServiceStartType uint32

const (
	ServiceAutoStart ServiceStartType = 0x2

	ServiceBootStart ServiceStartType = 0x0

	ServiceDemandStart ServiceStartType = 0x3

	ServiceDisabled ServiceStartType = 0x4

	ServiceSystemStart ServiceStartType = 0x1
)

type ProcessAccessRight uint32

const (
	ProcessAllAccess             ProcessAccessRight = 0x1f0fff
	ProcessCreateProcess         ProcessAccessRight = 0x80
	ProcessCreateThread          ProcessAccessRight = 0x2
	ProcessDupHandle             ProcessAccessRight = 0x40
	ProcessQueryInformation      ProcessAccessRight = 0x400
	ProcessQueryLimitInformation ProcessAccessRight = 0x1000
	ProcessSetInformation        ProcessAccessRight = 0x200
	ProcessSetQuota              ProcessAccessRight = 0x100
	ProcessSuspendResume         ProcessAccessRight = 0x800
	ProcessTerminate             ProcessAccessRight = 0x1
	ProcessVmOperation           ProcessAccessRight = 0x8
	ProcessVmRead                ProcessAccessRight = 0x10
	ProcessVmWrite               ProcessAccessRight = 0x20
	ProcessSynchronize           ProcessAccessRight = 0x100000
)

type ServiceStatusProcess struct {
	DwServiceType             uint32
	DwCurrentState            uint32
	DwControlsAccepted        uint32
	DwWin32ExitCode           uint32
	DwServiceSpecificExitCode uint32
	DwCheckPoint              uint32
	DwWaitHint                uint32
	DwProcessId               uint32
	DwServiceFlags            uint32
}

type EnumServiceStatusProcess struct {
	LpServiceName        *int8
	LpDisplayName        *int8
	ServiceStatusProcess ServiceStatusProcess
	Pad_cgo_0            [4]byte
}

type QueryServiceConfig struct {
	DwServiceType      uint32
	DwStartType        uint32
	DwErrorControl     uint32
	Pad_cgo_0          [4]byte
	LpBinaryPathName   *int8
	LpLoadOrderGroup   *int8
	DwTagId            uint32
	Pad_cgo_1          [4]byte
	LpDependencies     *int8
	LpServiceStartName *int8
	LpDisplayName      *int8
}
