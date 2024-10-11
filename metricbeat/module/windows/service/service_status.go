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

package service

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"errors"

	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/gosigar"
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
	serviceStates = map[ServiceState]string{
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

func (startType ServiceStartType) String() string {
	return serviceStartTypes[startType]
}

func (state ServiceState) String() string {
	if val, ok := serviceStates[state]; ok {
		return val
	}
	return ""
}

func GetServiceStates(handle windows.Handle, state uint32, protectedServices map[string]struct{}) ([]Status, error) {
	var servicesReturned uint32
	var bytesNeeded uint32

	// fetch bytes needed
	if err := windows.EnumServicesStatusEx(handle, windows.SC_ENUM_PROCESS_INFO, windows.SERVICE_WIN32, state, nil, 0, &bytesNeeded, nil, nil, nil); err != nil && err != windows.ERROR_MORE_DATA {
		return nil, fmt.Errorf("error while calling the EnumServicesStatusEx api: %w", err)
	}

	buffer := make([]byte, bytesNeeded)

	for {
		if err := windows.EnumServicesStatusEx(handle, windows.SC_ENUM_PROCESS_INFO, windows.SERVICE_WIN32, state, &buffer[0], uint32(len(buffer)), &bytesNeeded, &servicesReturned, nil, nil); err != nil {
			if err == windows.ERROR_MORE_DATA {
				// Increase buffer size and retry.
				buffer = make([]byte, len(buffer)+int(bytesNeeded))
				continue
			}
			return nil, fmt.Errorf("error while calling the EnumServicesStatusEx api: %w", err)
		}

		break
	}

	processes := unsafe.Slice((*windows.ENUM_SERVICE_STATUS_PROCESS)(unsafe.Pointer(&buffer[0])), int(servicesReturned))

	var services []Status
	for _, proc := range processes {
		service, err := getServiceInformation(proc, handle, protectedServices)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

func getServiceInformation(rawService windows.ENUM_SERVICE_STATUS_PROCESS, handle windows.Handle, protectedServices map[string]struct{}) (Status, error) {
	service := Status{
		PID: rawService.ServiceStatusProcess.ProcessId,
	}

	service.DisplayName = windows.UTF16PtrToString(rawService.DisplayName)
	service.ServiceName = windows.UTF16PtrToString(rawService.ServiceName)

	var state string

	if stat, ok := serviceStates[ServiceState(rawService.ServiceStatusProcess.CurrentState)]; ok {
		state = stat
	} else {
		state = "Can not define State"
	}
	service.CurrentState = state

	// Exit code.
	service.ExitCode = rawService.ServiceStatusProcess.Win32ExitCode
	if service.ExitCode == uint32(windows.ERROR_SERVICE_SPECIFIC_ERROR) {
		service.ExitCode = rawService.ServiceStatusProcess.ServiceSpecificExitCode
	}

	serviceHandle, err := windows.OpenService(handle, rawService.ServiceName, windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return service, fmt.Errorf("error while opening service %s: %w", service.ServiceName, err)
	}

	defer windows.CloseHandle(serviceHandle)

	// Get detailed information
	if err := getAdditionalServiceInfo(serviceHandle, &service); err != nil {
		return service, err
	}

	// Get optional information
	if err := getOptionalServiceInfo(serviceHandle, &service); err != nil {
		return service, err
	}

	//Get uptime for service
	if ServiceState(rawService.ServiceStatusProcess.CurrentState) != ServiceStopped {
		processUpTime, err := getServiceUptime(rawService.ServiceStatusProcess.ProcessId)
		if err != nil {
			if _, ok := protectedServices[service.ServiceName]; errors.Is(err, os.ErrPermission) && !ok {
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

func getAdditionalServiceInfo(serviceHandle windows.Handle, service *Status) error {
	var bytesNeeded uint32

	// fetch bytes needed
	if err := windows.QueryServiceConfig(serviceHandle, nil, 0, &bytesNeeded); err != windows.ERROR_INSUFFICIENT_BUFFER {
		return fmt.Errorf("error while querying the service configuration %s: %w", service.ServiceName, err)
	}

	buffer := make([]byte, bytesNeeded)
	serviceQueryConfig := (*windows.QUERY_SERVICE_CONFIG)(unsafe.Pointer(&buffer[0]))

	for {
		if err := windows.QueryServiceConfig(serviceHandle, serviceQueryConfig, uint32(len(buffer)), &bytesNeeded); err != nil {
			if err == windows.ERROR_INSUFFICIENT_BUFFER {
				// Increase buffer size and retry.
				buffer = make([]byte, len(buffer)+int(bytesNeeded))
				continue
			}
			return fmt.Errorf("error while querying the service configuration %s: %w", service.ServiceName, err)
		}
		service.StartType = ServiceStartType(serviceQueryConfig.StartType)
		service.ServiceStartName = windows.UTF16PtrToString(serviceQueryConfig.ServiceStartName)
		service.BinaryPathName = windows.UTF16PtrToString(serviceQueryConfig.BinaryPathName)

		break
	}

	return nil
}

func getOptionalServiceInfo(serviceHandle windows.Handle, service *Status) error {
	// Get information if the service is started delayed or triggered. Only valid for automatic or manual services. So filter them first.
	if service.StartType == StartTypeAutomatic || service.StartType == StartTypeManual {
		var delayedInfo *windows.SERVICE_DELAYED_AUTO_START_INFO
		if service.StartType == StartTypeAutomatic {
			delayedInfoBuffer, err := queryServiceConfig2(serviceHandle, windows.SERVICE_CONFIG_DELAYED_AUTO_START_INFO)
			if err != nil {
				return fmt.Errorf("error while querying the service configuration2 %s: %w", service.ServiceName, err)
			}

			delayedInfo = (*windows.SERVICE_DELAYED_AUTO_START_INFO)(unsafe.Pointer(&delayedInfoBuffer[0]))
		}

		// Get information if the service is triggered.
		triggeredInfoBuffer, err := queryServiceConfig2(serviceHandle, windows.SERVICE_CONFIG_TRIGGER_INFO)
		if err != nil {
			return fmt.Errorf("error while querying the service configuration2 trigger %s: %w", service.ServiceName, err)
		}

		triggeredInfo := (*serviceTriggerInfo)(unsafe.Pointer(&triggeredInfoBuffer[0]))

		if service.StartType == StartTypeAutomatic {
			if triggeredInfo.cTriggers > 0 && delayedInfo.IsDelayedAutoStartUp != 0 {
				service.StartType = StartTypeAutomaticDelayedTriggered
			} else if triggeredInfo.cTriggers > 0 {
				service.StartType = StartTypeAutomaticTriggered
			} else if delayedInfo.IsDelayedAutoStartUp != 0 {
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

func queryServiceConfig2(serviceHandle windows.Handle, infoLevel uint32) ([]byte, error) {
	var bytesNeeded uint32

	// fetch bytes needed
	if err := windows.QueryServiceConfig2(serviceHandle, infoLevel, nil, 0, &bytesNeeded); err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("error while calling the QueryServiceConfig2 api: %w", err)
	}

	buffer := make([]byte, bytesNeeded)

	for {
		if err := windows.QueryServiceConfig2(serviceHandle, uint32(infoLevel), &buffer[0], uint32(len(buffer)), &bytesNeeded); err != nil {
			if err == windows.ERROR_INSUFFICIENT_BUFFER {
				// Increase buffer size and retry.
				buffer = make([]byte, len(buffer)+int(bytesNeeded))
				continue
			}
			return nil, fmt.Errorf("error while calling the QueryServiceConfig2 api: %w", err)
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
