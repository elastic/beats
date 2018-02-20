// +build windows

package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys"
	"github.com/elastic/gosigar"
)

// Windows API calls
//sys _OpenSCManager(machineName *uint16, databaseName *uint16, desiredAcces ServiceSCMAccessRight) (handle ServiceDatabaseHandle, err error) = advapi32.OpenSCManagerW
//sys _EnumServicesStatusEx(handle ServiceDatabaseHandle, infoLevel ServiceInfoLevel, serviceType ServiceType, serviceState ServiceEnumState, services *byte, bufSize uint32, bytesNeeded *uint32, servicesReturned *uint32, resumeHandle *uintptr, groupName *uintptr) (err error) [failretval==0] = advapi32.EnumServicesStatusExW
//sys _OpenService(handle ServiceDatabaseHandle, serviceName *uint16, desiredAccess ServiceAccessRight) (serviceHandle ServiceHandle, err error) = advapi32.OpenServiceW
//sys _QueryServiceConfig(serviceHandle ServiceHandle, serviceConfig *byte, bufSize uint32, bytesNeeded *uint32) (err error) [failretval==0] = advapi32.QueryServiceConfigW
//sys _QueryServiceConfig2(serviceHandle ServiceHandle, infoLevel ServiceConfigInformation, configBuffer *byte, bufSize uint32, bytesNeeded *uint32) (err error) [failretval==0] = advapi32.QueryServiceConfig2W
//sys _CloseServiceHandle(handle uintptr) (err error) = advapi32.CloseServiceHandle

var (
	sizeofEnumServiceStatusProcess = (int)(unsafe.Sizeof(EnumServiceStatusProcess{}))
)

type ServiceDatabaseHandle uintptr

type ServiceHandle uintptr

type ProcessHandle uintptr

type ServiceConfigInformation uint32

const (
	ServiceConfigDelayedAutoStartInfo   ServiceConfigInformation = 3
	ServiceConfigDescription            ServiceConfigInformation = 1
	ServiceConfigFailureActions         ServiceConfigInformation = 2
	ServiceConfigFailureActionsFlag     ServiceConfigInformation = 4
	ServiceConfigPreferredNode          ServiceConfigInformation = 9
	ServiceConfigPreshutdownInfo        ServiceConfigInformation = 7
	ServiceConfigRequiredPrivilegesInfo ServiceConfigInformation = 6
	ServiceConfigServiceSidInfo         ServiceConfigInformation = 5
	ServiceConfigTriggerInfo            ServiceConfigInformation = 8
	ServiceConfigLaunchProtected        ServiceConfigInformation = 12
)

type serviceDelayedAutoStartInfo struct {
	delayedAutoStart bool
}

type serviceTriggerInfo struct {
	cTriggers uint32
	pTriggers uintptr
	pReserved uintptr
}

var serviceStates = map[ServiceState]string{
	ServiceContinuePending: "Continuing",
	ServicePausePending:    "Pausing",
	ServicePaused:          "Paused",
	ServiceRunning:         "Running",
	ServiceStartPending:    "Starting",
	ServiceStopPending:     "Stopping",
	ServiceStopped:         "Stopped",
}

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

var serviceStartTypes = map[ServiceStartType]string{
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

func (startType ServiceStartType) String() string {
	return serviceStartTypes[startType]
}

func (state ServiceState) String() string {
	if val, ok := serviceStates[state]; ok {
		return val
	}
	return ""
}

// errorNames is mapping of errno values to names.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681383(v=vs.85).aspx
var errorNames = map[uint32]string{
	1077: "ERROR_SERVICE_NEVER_STARTED",
}

type ServiceStatus struct {
	DisplayName  string
	ServiceName  string
	CurrentState string
	StartType    ServiceStartType
	PID          uint32 // ID of the associated process.
	Uptime       time.Duration
	ExitCode     uint32 // Exit code for stopped services.
}

type ServiceReader struct {
	handle            ServiceDatabaseHandle
	state             ServiceEnumState
	guid              string            // Host's MachineGuid value (a unique ID for the host).
	ids               map[string]string // Cache of service IDs.
	protectedServices map[string]struct{}
}

var InvalidServiceDatabaseHandle = ^ServiceDatabaseHandle(0)
var InvalidServiceHandle = ^ServiceHandle(0)

func OpenSCManager(machineName string, databaseName string, desiredAccess ServiceSCMAccessRight) (ServiceDatabaseHandle, error) {
	var machineNamePtr *uint16
	if machineName != "" {
		var err error
		machineNamePtr, err = syscall.UTF16PtrFromString(machineName)
		if err != nil {
			return InvalidServiceDatabaseHandle, err
		}
	}

	var databaseNamePtr *uint16
	if databaseName != "" {
		var err error
		databaseNamePtr, err = syscall.UTF16PtrFromString(databaseName)
		if err != nil {
			return InvalidServiceDatabaseHandle, err
		}
	}

	handle, err := _OpenSCManager(machineNamePtr, databaseNamePtr, desiredAccess)
	if err != nil {
		return InvalidServiceDatabaseHandle, ServiceErrno(err.(syscall.Errno))
	}

	return handle, nil
}

func OpenService(handle ServiceDatabaseHandle, serviceName string, desiredAccess ServiceAccessRight) (ServiceHandle, error) {
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

func QueryServiceConfig2(serviceHandle ServiceHandle, infoLevel ServiceConfigInformation) ([]byte, error) {
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

func getServiceStates(handle ServiceDatabaseHandle, state ServiceEnumState, protectedServices map[string]struct{}) ([]ServiceStatus, error) {
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
			return nil, ServiceErrno(err.(syscall.Errno))
		}

		break
	}

	// Windows appears to tack on a single byte null terminator to the UTF-16
	// strings, but we are expecting either no null terminator or \u0000 (an
	// even number of bytes).
	if len(servicesBuffer)%2 != 0 && servicesBuffer[len(servicesBuffer)-1] == 0 {
		servicesBuffer = servicesBuffer[:len(servicesBuffer)-1]
	}

	var services []ServiceStatus
	for i := 0; i < int(servicesReturned); i++ {
		serviceTemp := (*EnumServiceStatusProcess)(unsafe.Pointer(&servicesBuffer[i*sizeofEnumServiceStatusProcess]))

		service, err := getServiceInformation(serviceTemp, servicesBuffer, handle, protectedServices)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

func getServiceInformation(rawService *EnumServiceStatusProcess, servicesBuffer []byte, handle ServiceDatabaseHandle, protectedServices map[string]struct{}) (ServiceStatus, error) {
	service := ServiceStatus{
		PID: rawService.ServiceStatusProcess.DwProcessId,
	}

	// Read null-terminated UTF16 strings from the buffer.
	serviceNameOffset := uintptr(unsafe.Pointer(rawService.LpServiceName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))
	displayNameOffset := uintptr(unsafe.Pointer(rawService.LpDisplayName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))

	strBuf := new(bytes.Buffer)
	if err := sys.UTF16ToUTF8Bytes(servicesBuffer[displayNameOffset:], strBuf); err != nil {
		return service, err
	}
	service.DisplayName = strBuf.String()

	strBuf.Reset()
	if err := sys.UTF16ToUTF8Bytes(servicesBuffer[serviceNameOffset:], strBuf); err != nil {
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

	serviceHandle, err := OpenService(handle, service.ServiceName, ServiceQueryConfig)
	if err != nil {
		return service, err
	}

	defer CloseServiceHandle(serviceHandle)

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

func getAdditionalServiceInfo(serviceHandle ServiceHandle, service *ServiceStatus) error {
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
			return ServiceErrno(err.(syscall.Errno))
		}
		serviceQueryConfig := (*QueryServiceConfig)(unsafe.Pointer(&buffer[0]))
		service.StartType = ServiceStartType(serviceQueryConfig.DwStartType)
		break
	}

	return nil
}

func getOptionalServiceInfo(serviceHandle ServiceHandle, service *ServiceStatus) error {
	// Get information if the service is started delayed or triggered. Only valid for automatic or manual services. So filter them first.
	if service.StartType == StartTypeAutomatic || service.StartType == StartTypeManual {
		var delayedInfo *serviceDelayedAutoStartInfo
		if service.StartType == StartTypeAutomatic {
			delayedInfoBuffer, err := QueryServiceConfig2(serviceHandle, ServiceConfigDelayedAutoStartInfo)
			if err != nil {
				return err
			}

			delayedInfo = (*serviceDelayedAutoStartInfo)(unsafe.Pointer(&delayedInfoBuffer[0]))
		}

		// Get information if the service is triggered.
		triggeredInfoBuffer, err := QueryServiceConfig2(serviceHandle, ServiceConfigTriggerInfo)
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

func (reader *ServiceReader) Close() error {
	return CloseServiceDatabaseHandle(reader.handle)
}

func CloseServiceDatabaseHandle(handle ServiceDatabaseHandle) error {
	if err := _CloseServiceHandle(uintptr(handle)); err != nil {
		return ServiceErrno(err.(syscall.Errno))
	}

	return nil
}

func CloseServiceHandle(handle ServiceHandle) error {
	if err := _CloseServiceHandle(uintptr(handle)); err != nil {
		return ServiceErrno(err.(syscall.Errno))
	}

	return nil
}

func NewServiceReader() (*ServiceReader, error) {
	hndl, err := OpenSCManager("", "", ScManagerEnumerateService|ScManagerConnect)
	if err != nil {
		return nil, errors.Wrap(err, "initialization failed")
	}

	guid, err := getMachineGUID()
	if err != nil {
		return nil, err
	}

	r := &ServiceReader{
		handle:            hndl,
		state:             ServiceStateAll,
		guid:              guid,
		ids:               map[string]string{},
		protectedServices: map[string]struct{}{},
	}

	return r, nil
}

func (reader *ServiceReader) Read() ([]common.MapStr, error) {
	services, err := getServiceStates(reader.handle, reader.state, reader.protectedServices)
	if err != nil {
		return nil, err
	}

	result := make([]common.MapStr, 0, len(services))

	for _, service := range services {
		ev := common.MapStr{
			"id":           reader.getServiceID(service.ServiceName),
			"display_name": service.DisplayName,
			"name":         service.ServiceName,
			"state":        service.CurrentState,
			"start_type":   service.StartType.String(),
		}

		if service.CurrentState == "Stopped" {
			ev.Put("exit_code", getErrorCode(service.ExitCode))
		}

		if service.PID > 0 {
			ev.Put("pid", service.PID)
		}

		if service.Uptime > 0 {
			if _, err = ev.Put("uptime.ms", service.Uptime); err != nil {
				return nil, err
			}
		}

		result = append(result, ev)
	}

	return result, nil
}

// getServiceID returns a unique ID for the service that is derived from the
// machine's GUID and the service's name.
func (reader *ServiceReader) getServiceID(name string) string {
	// hash returns a base64 encoded sha256 hash that is truncated to 10 chars.
	hash := func(v string) string {
		sum := sha256.Sum256([]byte(v))
		base64Hash := base64.RawURLEncoding.EncodeToString(sum[:])
		return base64Hash[:10]
	}

	id, found := reader.ids[name]
	if !found {
		id = hash(reader.guid + name)
		reader.ids[name] = id
	}

	return id
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

// getMachineGUID returns the machine's GUID value which is unique to a Windows
// installation.
func getMachineGUID() (string, error) {
	const key = registry.LOCAL_MACHINE
	const path = `SOFTWARE\Microsoft\Cryptography`
	const name = "MachineGuid"

	k, err := registry.OpenKey(key, path, registry.READ)
	if err != nil {
		return "", errors.Wrapf(err, `failed to open HKLM\%v`, path)
	}

	guid, _, err := k.GetStringValue(name)
	if err != nil {
		return "", errors.Wrapf(err, `failed to get value of HKLM\%v\%v`, path, name)
	}

	return guid, nil
}

func getErrorCode(errno uint32) string {
	name, found := errorNames[errno]
	if found {
		return name
	}
	return strconv.Itoa(int(errno))
}
