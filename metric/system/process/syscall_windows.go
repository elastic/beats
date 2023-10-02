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

package process

// Windows Process Snapshotting docs:
//  - https://learn.microsoft.com/en-us/previous-versions/windows/desktop/proc_snap/overview-of-process-snapshotting
// PssCaptureSnapshot docs in https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/nf-processsnapshot-psscapturesnapshot
// PssQuerySnapshot docs in https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/nf-processsnapshot-pssquerysnapshot

// Use golang.org/x/sys/windows/mkwinsyscall instead of adriansr/mksyscall
// below once https://github.com/golang/go/issues/42373 is fixed.
//go:generate go get github.com/adriansr/mkwinsyscall
//go:generate $GOPATH/bin/mkwinsyscall.exe -systemdll -output zsyscall_windows.go syscall_windows.go

//sys PssCaptureSnapshot(processHandle syscall.Handle, captureFlags PSSCaptureFlags, threadContextFlags uint32, snapshotHandle *syscall.Handle) (err error) [failretval!=0] = kernel32.PssCaptureSnapshot
//sys PssQuerySnapshot(snapshotHandle syscall.Handle, informationClass uint32, buffer *PssThreadInformation, bufferLength uint32) (err error) [failretval!=0] = kernel32.PssQuerySnapshot

// The following constants are PssQueryInformationClass as defined on
// https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/ne-processsnapshot-pss_query_information_class
const (
	PssQueryProcessInformation uint32 = iota
	PssQueryVaCloneInformation
	PssQueryAuxiliaryPagesInformation
	PssQueryVaSpaceInformation
	PssQueryHandleInformation
	PssQueryThreadInformation
	PssQueryHandleTraceInformation
	PssQueryPerformanceCounters
)

// PSSCaptureFlags from
// https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/ne-processsnapshot-pss_capture_flags
type PSSCaptureFlags uint32

const (
	PSSCaptureNone                          PSSCaptureFlags = 0x00000000
	PSSCaptureVAClone                       PSSCaptureFlags = 0x00000001
	PSSCaptureReserved00000002              PSSCaptureFlags = 0x00000002
	PSSCaptureHandles                       PSSCaptureFlags = 0x00000004
	PSSCaptureHandleNameInformation         PSSCaptureFlags = 0x00000008
	PSSCaptureHandleBasicInformation        PSSCaptureFlags = 0x00000010
	PSSCaptureHandleTypeSpecificInformation PSSCaptureFlags = 0x00000020
	PSSCaptureHandleTrace                   PSSCaptureFlags = 0x00000040
	PSSCaptureThreads                       PSSCaptureFlags = 0x00000080
	PSSCaptureThreadContext                 PSSCaptureFlags = 0x00000100
	PSSCaptureThreadContextExtended         PSSCaptureFlags = 0x00000200
	PSSCaptureReserved00000400              PSSCaptureFlags = 0x00000400
	PSSCaptureVASpace                       PSSCaptureFlags = 0x00000800
	PSSCaptureVASpaceSectionInformation     PSSCaptureFlags = 0x00001000
	PSSCaptureIPTTrace                      PSSCaptureFlags = 0x00002000
	PSSCaptureReserved00004000              PSSCaptureFlags = 0x00004000
	PSSCreateBreakawayOptional              PSSCaptureFlags = 0x04000000
	PSSCreateBreakaway                      PSSCaptureFlags = 0x08000000
	PSSCreateForceBreakaway                 PSSCaptureFlags = 0x10000000
	PSSCreateUseVMAllocations               PSSCaptureFlags = 0x20000000
	PSSCreateMeasurePerformance             PSSCaptureFlags = 0x40000000
	PSSCreateReleaseSection                 PSSCaptureFlags = 0x80000000
)

type PssThreadInformation struct {
	ThreadsCaptured uint32
	ContextLength   uint32
}
