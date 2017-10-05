// +build windows

package services

import (
	"bytes"
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
	"github.com/pkg/errors"

	"github.com/elastic/beats/winlogbeat/sys"
	"golang.org/x/sys/windows"
)

// Windows API calls
//sys _OpenSCManager(machineName *uint16, databaseName *uint16, desiredAcces ServiceSCMAccessRight) (handle ServiceDatabaseHandle, err error) = advapi32.OpenSCManagerW
//sys _EnumServicesStatusEx(handle ServiceDatabaseHandle, infoLevel ServiceInfoLevel, serviceType ServiceType, serviceState ServiceEnumState, services *byte, bufSize uint32, bytesNeeded *uint32, servicesReturned *uint32, resumeHandle *uintptr, groupName *uintptr) (err error) [failretval==0] = advapi32.EnumServicesStatusExW
//sys _OpenService(handle ServiceDatabaseHandle, serviceName *uint16, desiredAccess ServiceAccessRight) (serviceHandle ServiceHandle, err error) = advapi32.OpenServiceW
//sys _QueryServiceConfig(serviceHandle ServiceHandle, serviceConfig *byte, bufSize uint32, bytesNeeded *uint32) (err error) [failretval==0] = advapi32.QueryServiceConfigW
//sys _CloseServiceHandle(handle uintptr) (err error) = advapi32.CloseServiceHandle

var (
	sizeOfEnumServiceStatusProcess = (int)(unsafe.Sizeof(EnumServiceStatusProcess{}))
	sizeOfQueryServiceConfig       = (int)(unsafe.Sizeof(QueryServiceConfig{}))
)

type enumServiceStatusProcess struct {
	LpServiceName        uintptr
	LpDisplayName        uintptr
	ServiceStatusProcess ServiceStatusProcess
}

type ServiceDatabaseHandle uintptr

type ServiceHandle uintptr

var serviceStates = map[ServiceState]string{
	ServiceContinuePending: "ServiceContinuePending",
	ServicePausePending:    "ServicePausePending",
	ServicePaused:          "ServicePaused",
	ServiceRuning:          "ServiceRuning",
	ServiceStartPending:    "ServiceStartPending",
	ServiceStopPending:     "ServiceStopPending",
	ServiceStopped:         "ServiceStopped",
}

var serviceStartTypes = map[ServiceStartType]string{
	ServiceAutoStart:   "ServiceAutoStart",
	ServiceBootStart:   "ServiceBootStart",
	ServiceDemandStart: "ServiceDemandStart",
	ServiceDisabled:    "ServiceDisabled",
	ServiceSystemStart: "ServiceSystemStart",
}

func (state ServiceState) String() string {
	if val, ok := serviceStates[state]; ok {
		return val
	}
	return ""
}

type ServiceStatus struct {
	DisplayName  string
	ServiceName  string
	CurrentState string
	StartType    string
}

type ServiceReader struct {
	handle ServiceDatabaseHandle
	state  ServiceEnumState
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

func getServiceStates(handle ServiceDatabaseHandle, state ServiceEnumState) ([]ServiceStatus, error) {
	var bufSize uint32
	var bytesNeeded uint32
	var servicesReturned uint32
	var lastOffset uintptr
	var services []ServiceStatus

	if err := _EnumServicesStatusEx(handle, ScEnumProcessInfo, ServiceWin32, state, nil, bufSize, &bytesNeeded, &servicesReturned, nil, nil); err != nil {
		if ServiceErrno(err.(syscall.Errno)) != SERVICE_ERROR_MORE_DATA {
			return services, ServiceErrno(err.(syscall.Errno))
		}
		bufSize += bytesNeeded
		servicesBuffer := make([]byte, bytesNeeded)
		lastOffset = uintptr(len(servicesBuffer)) - 1

		// This loop should not repeat more then two times
		for {
			if err := _EnumServicesStatusEx(handle, ScEnumProcessInfo, ServiceWin32, state, &servicesBuffer[0], bufSize, &bytesNeeded, &servicesReturned, nil, nil); err != nil {
				if ServiceErrno(err.(syscall.Errno)) != SERVICE_ERROR_MORE_DATA {
					return nil, ServiceErrno(err.(syscall.Errno))
				}
				bufSize += bytesNeeded
			} else {

				displayNameBuffer := new(bytes.Buffer)
				serviceNameBuffer := new(bytes.Buffer)

				for i := 0; i < int(servicesReturned); i++ {
					serviceTemp := (*EnumServiceStatusProcess)(unsafe.Pointer(&servicesBuffer[i*sizeOfEnumServiceStatusProcess]))

					serviceNameOffset := uintptr(unsafe.Pointer(serviceTemp.LpServiceName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))
					displayNameOffset := uintptr(unsafe.Pointer(serviceTemp.LpDisplayName)) - (uintptr)(unsafe.Pointer(&servicesBuffer[0]))

					displayNameBuffer.Reset()
					serviceNameBuffer.Reset()

					if err = sys.UTF16ToUTF8Bytes(servicesBuffer[displayNameOffset:serviceNameOffset], displayNameBuffer); err != nil {
						return nil, err
					}

					if err = sys.UTF16ToUTF8Bytes(servicesBuffer[serviceNameOffset:lastOffset], serviceNameBuffer); err != nil {
						return nil, err
					}

					lastOffset = displayNameOffset

					service := ServiceStatus{}

					service.DisplayName = displayNameBuffer.String()
					service.ServiceName = serviceNameBuffer.String()

					var state string

					if stat, ok := serviceStates[ServiceState(serviceTemp.ServiceStatusProcess.DwCurrentState)]; ok {
						state = stat
					} else {
						state = "Can not define State"
					}
					service.CurrentState = state

					// Get detailed information
					var serviceBufSize uint32
					var serviceBytesNeeded uint32

					serviceHandle, err := OpenService(handle, service.ServiceName, ServiceQueryConfig)
					if err != nil {
						return nil, err
					}

					if err := _QueryServiceConfig(serviceHandle, nil, serviceBufSize, &serviceBytesNeeded); err != nil {
						if ServiceErrno(err.(syscall.Errno)) != SERVICE_ERROR_INSUFFICIENT_BUFFER {
							if err := CloseServiceHandle(serviceHandle); err != nil {
								return nil, err
							}
							return nil, err
						}
						serviceBufSize += serviceBytesNeeded
						buffer := make([]byte, serviceBufSize)

						for {
							if err := _QueryServiceConfig(serviceHandle, &buffer[0], serviceBufSize, &serviceBytesNeeded); err != nil {
								if ServiceErrno(err.(syscall.Errno)) != SERVICE_ERROR_INSUFFICIENT_BUFFER {
									if err := CloseServiceHandle(serviceHandle); err != nil {
										return nil, err
									}
									return nil, err
								}
								serviceBufSize += serviceBytesNeeded
							} else {
								serviceQueryConfig := (*QueryServiceConfig)(unsafe.Pointer(&buffer[0]))
								service.StartType = serviceStartTypes[ServiceStartType(serviceQueryConfig.DwStartType)]
								if err := CloseServiceHandle(serviceHandle); err != nil {
									return nil, err
								}
								break
							}
						}
					}

					services = append(services, service)
				}

				return services, nil
			}
		}
	}

	return nil, nil
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

	r := &ServiceReader{
		handle: hndl,
	}

	//var state ServiceEnumState

	// configState := strings.ToLower(config.State)
	// switch configState {
	// case "all", "":
	// 	state = ServiceStateAll
	// case "active":
	// 	state = ServiceActive
	// case "inactive":
	// 	state = ServiceInActive
	// default:
	// 	err := fmt.Errorf("state '%s' are not valid", configState)
	// 	r.Close()
	// 	return nil, errors.Wrap(err, "initialization failed")
	// }

	r.state = ServiceStateAll

	return r, nil
}

func (reader *ServiceReader) Read() ([]common.MapStr, error) {

	services, err := getServiceStates(reader.handle, reader.state)

	if err != nil {
		return nil, err
	}

	result := make([]common.MapStr, 0, len(services))

	for _, service := range services {
		ev := common.MapStr{
			"display_name": service.DisplayName,
			"service_name": service.ServiceName,
			"state":        service.CurrentState,
			"start_type":   service.StartType,
		}

		result = append(result, ev)
	}

	return result, nil
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
