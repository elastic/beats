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

package service

import (
	"bytes"
	"strconv"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/gosigar"
)

// Windows API calls
//sys _OpenSCManager(machineName *uint16, databaseName *uint16, desiredAcces ServiceSCMAccessRight) (handle ServiceDatabaseHandle, err error) = advapi32.OpenSCManagerW
//sys _EnumServicesStatusEx(handle ServiceDatabaseHandle, infoLevel ServiceInfoLevel, serviceType ServiceType, serviceState ServiceEnumState, services *byte, bufSize uint32, bytesNeeded *uint32, servicesReturned *uint32, resumeHandle *uintptr, groupName *uintptr) (err error) [failretval==0] = advapi32.EnumServicesStatusExW
//sys _OpenService(handle ServiceDatabaseHandle, serviceName *uint16, desiredAccess ServiceAccessRight) (serviceHandle ServiceHandle, err error) = advapi32.OpenServiceW
//sys _QueryServiceConfig(serviceHandle ServiceHandle, serviceConfig *byte, bufSize uint32, bytesNeeded *uint32) (err error) [failretval==0] = advapi32.QueryServiceConfigW
//sys _QueryServiceConfig2(serviceHandle ServiceHandle, infoLevel ServiceConfigInformation, configBuffer *byte, bufSize uint32, bytesNeeded *uint32) (err error) [failretval==0] = advapi32.QueryServiceConfig2W
//sys _CloseServiceHandle(handle uintptr) (err error) = advapi32.CloseServiceHandle

const (
	ConfigDelayedAutoStartInfo   ConfigInformation = 3
	ConfigTriggerInfo            ConfigInformation = 8
	ConfigLaunchProtected        ConfigInformation = 12
	ConfigDescription            ConfigInformation = 1
	ConfigFailureActions         ConfigInformation = 2
	ConfigFailureActionsFlag     ConfigInformation = 4
	ConfigPreferredNode          ConfigInformation = 9
	ConfigPreshutdownInfo        ConfigInformation = 7
	ConfigRequiredPrivilegesInfo ConfigInformation = 6
	ConfigServiceSidInfo         ConfigInformation = 5
)

const (
	StartTypeBoot ServiceStartType = iota
	StartTypeSystem
	StartTypeAutomatic
	StartTypeManual
	StartTypeDisabled
	StartTypeAutomaticDelayed
	StartTypeAutomaticTriggered
	StartTypeAutomaticDelayedTriggered
	StartTypeManualTriggered
)

var (
	InvalidServiceHandle = ^Handle(0)
	serviceStates        = map[ServiceState]string{
		ServiceContinuePending: "Continuing",
		ServicePausePending:    "Pausing",
		ServicePaused:          "Paused",
		ServiceRunning:         "Running",
		ServiceStartPending:    "Starting",
		ServiceStopPending:     "Stopping",
		ServiceStopped:         "Stopped",
	}
	serviceStartTypes = map[ServiceStartType]string{
		StartTypeBoot:                      "Boot",
		StartTypeSystem:                    "System",
		StartTypeAutomatic:                 "Automatic",
		StartTypeManual:                    "Manual",
		StartTypeDisabled:                  "Disabled",
		StartTypeAutomaticDelayed:          "Automatic (Delayed)",
		StartTypeAutomaticTriggered:        "Automatic (Triggered)",
		StartTypeAutomaticDelayedTriggered: "Automatic (Delayed, Triggered)",
		StartTypeManualTriggered:           "Manual (Triggered)",
	}
)

type ConfigInformation uint32

type Status struct {
	DisplayName      string
	ServiceName      string
	CurrentState     string
	StartType        ServiceStartType
	PID              uint32 // ID of the associated process.
	Uptime           time.Duration
	ExitCode         uint32 // Exit code for stopped services.
	ServiceStartName string
	BinaryPathName   string
}

type serviceTriggerInfo struct {
	cTriggers uint32
	pTriggers uintptr
	pReserved uintptr
}

type serviceDelayedAutoStartInfo struct {
	delayedAutoStart bool
}

func (startType ServiceStartType) String() string {
	return serviceStartTypes[startType]
}

func (state ServiceState) String() string {
	if val, ok := serviceStates[state]; ok {
		return val
	}
	return ""
}

func GetServiceStates(handle Handle, state ServiceEnumState, protectedServices map[string]struct{}) ([]Status, error) {
	var servicesReturned uint32
	var servicesBuffer []byte

	for {
		var bytesNeeded uint32
		var buf *byte
		if len(servicesBuffer) > 0 {
			buf = &servicesBuffer[0]
		}

		if err := _EnumServicesStatusEx(handle, ScEnumProcessInfo, ServiceWin32, state, buf, uint32(len(servicesBuffer)), &bytesNeeded, &servicesReturned, nil, nil); err != nil {
			if ServiceErrno(err.(syscall.Errno)) == SERVICE_ERROR_MORE_DATA {
				// Increase buffer size and retry.
				servicesBuffer = make([]byte, len(servicesBuffer)+int(bytesNeeded))
				continue
			}
			return nil, errors.Wrap(ServiceErrno(err.(syscall.Errno)), "error while calling the _EnumServicesStatusEx api")
		}

		break
	}

	// Windows appears to tack on a single byte null terminator to the UTF-16
	// strings, but we are expecting either no null terminator or \u0000 (an
	// even number of bytes).
	if len(servicesBuffer)%2 != 0 && servicesBuffer[len(servicesBuffer)-1] == 0 {
		servicesBuffer = servicesBuffer[:len(servicesBuffer)-1]
	}

	var services []Status
	var sizeStatusProcess = (int)(unsafe.Sizeof(EnumServiceStatusProcess{}))
	for i := 0; i < int(servicesReturned); i++ {
		serviceTemp := (*EnumServiceStatusProcess)(unsafe.Pointer(&servicesBuffer[i*sizeStatusProcess]))

		service, err := getServiceInformation(serviceTemp, servicesBuffer, handle, protectedServices)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

func getServiceInformation(rawService *EnumServiceStatusProcess, servicesBuffer []byte, handle Handle, protectedServices map[string]struct{}) (Status, error) {
	service := Status{
		PID: rawService.ServiceStatusProcess.DwProcessId,
	}

	// Read null-terminated UTF16 strings from the buffer.
	serviceNameOffset := uintptr(unsafe.Pointer(rawService.LpServiceName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))
	displayNameOffset := uintptr(unsafe.Pointer(rawService.LpDisplayName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))

	strBuf := new(bytes.Buffer)
	if err := common.UTF16ToUTF8Bytes(servicesBuffer[displayNameOffset:], strBuf); err != nil {
		return service, err
	}
	service.DisplayName = strBuf.String()

	strBuf.Reset()
	if err := common.UTF16ToUTF8Bytes(servicesBuffer[serviceNameOffset:], strBuf); err != nil {
		return service, err
	}
	service.ServiceName = strBuf.String()

	var state string

	if stat, ok := serviceStates[ServiceState(rawService.ServiceStatusProcess.DwCurrentState)]; ok {
		state = stat
	} else {
		state = "Can not define State"
	}
	service.CurrentState = state

	// Exit code.
	service.ExitCode = rawService.ServiceStatusProcess.DwWin32ExitCode
	if service.ExitCode == uint32(windows.ERROR_SERVICE_SPECIFIC_ERROR) {
		service.ExitCode = rawService.ServiceStatusProcess.DwServiceSpecificExitCode
	}

	serviceHandle, err := openServiceHandle(handle, service.ServiceName, ServiceQueryConfig)
	if err != nil {
		return service, errors.Wrapf(err, "error while opening service %s", service.ServiceName)
	}

	defer closeHandle(serviceHandle)

	// Get detailed information
	if err := getAdditionalServiceInfo(serviceHandle, &service); err != nil {
		return service, err
	}

	// Get optional information
	if err := getOptionalServiceInfo(serviceHandle, &service); err != nil {
		return service, err
	}

	//Get uptime for service
	if ServiceState(rawService.ServiceStatusProcess.DwCurrentState) != ServiceStopped {
		processUpTime, err := getServiceUptime(rawService.ServiceStatusProcess.DwProcessId)
		if err != nil {
			if _, ok := protectedServices[service.ServiceName]; errors.Cause(err) == syscall.ERROR_ACCESS_DENIED && !ok {
				protectedServices[service.ServiceName] = struct{}{}
				logp.Warn("Uptime for service %v is not available because of insufficient rights", service.ServiceName)
			} else {
				return service, err
			}
		}
		service.Uptime = processUpTime / time.Millisecond
	}

	return service, nil
}

func openServiceHandle(handle Handle, serviceName string, desiredAccess ServiceAccessRight) (Handle, error) {
	var serviceNamePtr *uint16
	if serviceName != "" {
		var err error
		serviceNamePtr, err = syscall.UTF16PtrFromString(serviceName)
		if err != nil {
			return InvalidServiceHandle, err
		}
	}

	serviceHandle, err := _OpenService(handle, serviceNamePtr, desiredAccess)
	if err != nil {
		return InvalidServiceHandle, ServiceErrno(err.(syscall.Errno))
	}

	return serviceHandle, nil
}

func getAdditionalServiceInfo(serviceHandle Handle, service *Status) error {
	var buffer []byte

	for {
		var bytesNeeded uint32
		var bufPtr *byte
		if len(buffer) > 0 {
			bufPtr = &buffer[0]
		}

		if err := _QueryServiceConfig(serviceHandle, bufPtr, uint32(len(buffer)), &bytesNeeded); err != nil {
			if ServiceErrno(err.(syscall.Errno)) == SERVICE_ERROR_INSUFFICIENT_BUFFER {
				// Increase buffer size and retry.
				buffer = make([]byte, len(buffer)+int(bytesNeeded))
				continue
			}
			return errors.Wrapf(ServiceErrno(err.(syscall.Errno)), "error while querying the service configuration %s", service.ServiceName)
		}
		serviceQueryConfig := (*QueryServiceConfig)(unsafe.Pointer(&buffer[0]))
		service.StartType = ServiceStartType(serviceQueryConfig.DwStartType)
		serviceStartNameOffset := uintptr(unsafe.Pointer(serviceQueryConfig.LpServiceStartName)) - (uintptr)(unsafe.Pointer(&buffer[0]))
		binaryPathNameOffset := uintptr(unsafe.Pointer(serviceQueryConfig.LpBinaryPathName)) - (uintptr)(unsafe.Pointer(&buffer[0]))

		strBuf := new(bytes.Buffer)
		if err := common.UTF16ToUTF8Bytes(buffer[serviceStartNameOffset:], strBuf); err != nil {
			return err
		}
		service.ServiceStartName = strBuf.String()

		strBuf.Reset()
		if err := common.UTF16ToUTF8Bytes(buffer[binaryPathNameOffset:], strBuf); err != nil {
			return err
		}
		service.BinaryPathName = strBuf.String()

		break
	}

	return nil
}

func getOptionalServiceInfo(serviceHandle Handle, service *Status) error {
	// Get information if the service is started delayed or triggered. Only valid for automatic or manual services. So filter them first.
	if service.StartType == StartTypeAutomatic || service.StartType == StartTypeManual {
		var delayedInfo *serviceDelayedAutoStartInfo
		if service.StartType == StartTypeAutomatic {
			delayedInfoBuffer, err := queryServiceConfig2(serviceHandle, ConfigDelayedAutoStartInfo)
			if err != nil {
				return errors.Wrapf(err, "error while querying rhe service configuration %s", service.ServiceName)
			}

			delayedInfo = (*serviceDelayedAutoStartInfo)(unsafe.Pointer(&delayedInfoBuffer[0]))
		}

		// Get information if the service is triggered.
		triggeredInfoBuffer, err := queryServiceConfig2(serviceHandle, ConfigTriggerInfo)
		if err != nil {
			return err
		}

		triggeredInfo := (*serviceTriggerInfo)(unsafe.Pointer(&triggeredInfoBuffer[0]))

		if service.StartType == StartTypeAutomatic {
			if triggeredInfo.cTriggers > 0 && delayedInfo.delayedAutoStart {
				service.StartType = StartTypeAutomaticDelayedTriggered
			} else if triggeredInfo.cTriggers > 0 {
				service.StartType = StartTypeAutomaticTriggered
			} else if delayedInfo.delayedAutoStart {
				service.StartType = StartTypeAutomaticDelayed
			}
			return nil
		}

		if service.StartType == StartTypeManual && triggeredInfo.cTriggers > 0 {
			service.StartType = StartTypeManualTriggered
		}
	}

	return nil
}

func queryServiceConfig2(serviceHandle Handle, infoLevel ConfigInformation) ([]byte, error) {
	var buffer []byte

	for {
		var bytesNeeded uint32
		var bufPtr *byte
		if len(buffer) > 0 {
			bufPtr = &buffer[0]
		}

		if err := _QueryServiceConfig2(serviceHandle, infoLevel, bufPtr, uint32(len(buffer)), &bytesNeeded); err != nil {
			if ServiceErrno(err.(syscall.Errno)) == SERVICE_ERROR_INSUFFICIENT_BUFFER {
				// Increase buffer size and retry.
				buffer = make([]byte, len(buffer)+int(bytesNeeded))
				continue
			}
			return nil, err
		}

		break
	}

	return buffer, nil
}

// getServiceUptime returns the uptime for process
func getServiceUptime(processID uint32) (time.Duration, error) {
	var processCreationTime gosigar.ProcTime

	err := processCreationTime.Get(int(processID))
	if err != nil {
		return time.Duration(processCreationTime.StartTime), err
	}

	uptime := time.Since(time.Unix(0, int64(processCreationTime.StartTime)*int64(time.Millisecond)))

	return uptime, nil
}

func (e ServiceErrno) Error() string {
	// If the value is not one of the known Service errors then assume its a
	// general windows error.
	if _, found := serviceErrors[e]; !found {
		return syscall.Errno(e).Error()
	}

	// Use FormatMessage to convert the service errno to a string.
	var flags uint32 = syscall.FORMAT_MESSAGE_FROM_SYSTEM | syscall.FORMAT_MESSAGE_ARGUMENT_ARRAY | syscall.FORMAT_MESSAGE_IGNORE_INSERTS
	b := make([]uint16, 300)
	n, err := windows.FormatMessage(flags, modadvapi32.Handle(), uint32(e), 0, b, nil)
	if err != nil {
		return "service error #" + strconv.Itoa(int(e))
	}

	// Trim terminating \r and \n
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n]))
}
