// +build ignore

package services

/*
#include <windows.h>
#cgo LDFLAGS: -ladvapi32
*/
import "C"

type ServiceErrno uintptr

// Service Error Codes
const (
	SERVICE_ERROR_ACCESS_DENIED           ServiceErrno = C.ERROR_ACCESS_DENIED
	SERVICE_ERROR_MORE_DATA               ServiceErrno = C.ERROR_MORE_DATA
	SERVICE_ERROR_INVALID_PARAMETER       ServiceErrno = C.ERROR_INVALID_PARAMETER
	SERVICE_ERROR_INVALID_HANDLE          ServiceErrno = C.ERROR_INVALID_HANDLE
	SERVICE_ERROR_INVALID_LEVEL           ServiceErrno = C.ERROR_INVALID_LEVEL
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS    ServiceErrno = C.ERROR_SHUTDOWN_IN_PROGRESS
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST ServiceErrno = C.ERROR_DATABASE_DOES_NOT_EXIST
)

var serviceErrors = map[ServiceErrno]struct{}{
	SERVICE_ERROR_ACCESS_DENIED:           struct{}{},
	SERVICE_ERROR_MORE_DATA:               struct{}{},
	SERVICE_ERROR_INVALID_PARAMETER:       struct{}{},
	SERVICE_ERROR_INVALID_HANDLE:          struct{}{},
	SERVICE_ERROR_INVALID_LEVEL:           struct{}{},
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS:    struct{}{},
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST: struct{}{},
}

type ServiceType uint32

// ServiceTypes
const (
	// Services of type SERVICE_KERNEL_DRIVER and SERVICE_FILE_SYSTEM_DRIVER.
	ServiceDriver ServiceType = C.SERVICE_DRIVER
	// File system driver services.
	ServiceFileSystemDriver ServiceType = C.SERVICE_FILE_SYSTEM_DRIVER
	// Driver services.
	ServiceKernelDriver ServiceType = C.SERVICE_KERNEL_DRIVER
	// Services of type SERVICE_WIN32_OWN_PROCESS and SERVICE_WIN32_SHARE_PROCESS.
	ServiceWin32 ServiceType = C.SERVICE_WIN32
	// Services that run in their own processes.
	ServiceWin32OwnProcess ServiceType = C.SERVICE_WIN32_OWN_PROCESS
	// Services that share a process with one or more other services.
	ServiceWin32Shareprocess ServiceType = C.SERVICE_WIN32_SHARE_PROCESS
)

type ServiceState uint32

// ServiceStates
const (
	ServiceContinuePending ServiceState = C.SERVICE_CONTINUE_PENDING
	ServicePausePending    ServiceState = C.SERVICE_PAUSE_PENDING
	ServicePaused          ServiceState = C.SERVICE_PAUSED
	ServiceRuning          ServiceState = C.SERVICE_RUNNING
	ServiceStartPending    ServiceState = C.SERVICE_START_PENDING
	ServiceStopPending     ServiceState = C.SERVICE_STOP_PENDING
	ServiceStopped         ServiceState = C.SERVICE_STOPPED
)

type ServiceEnumState uint32

//Service Enum States
const (
	// Enumerates services that are in the following states: SERVICE_START_PENDING, SERVICE_STOP_PENDING, SERVICE_RUNNING, SERVICE_CONTINUE_PENDING, SERVICE_PAUSE_PENDING, and SERVICE_PAUSED.
	ServiceActive ServiceEnumState = C.SERVICE_ACTIVE
	// Enumerates services that are in the SERVICE_STOPPED state.
	ServiceInActive ServiceEnumState = C.SERVICE_INACTIVE
	// Combines the SERVICE_ACTIVE and SERVICE_INACTIVE states.
	ServiceStateAll ServiceEnumState = C.SERVICE_STATE_ALL
)

type ServiceAccessRight uint32

// Service Access Rights
const (
	// Includes STANDARD_RIGHTS_REQUIRED, in addition to all access rights in this table.
	ScManagerAllAccess ServiceAccessRight = C.SC_MANAGER_ALL_ACCESS
	// Required to connect to the service control manager.
	ScManagerConnect ServiceAccessRight = C.SC_MANAGER_CONNECT
	// Required to call the EnumServicesStatus or EnumServicesStatusEx function to list the services that are in the database.
	ScManagerEnumerateService ServiceAccessRight = C.SC_MANAGER_ENUMERATE_SERVICE
	// Required to call the QueryServiceLockStatus function to retrieve the lock status information for the database.
	ScManagerQueryLockStatus ServiceAccessRight = C.SC_MANAGER_QUERY_LOCK_STATUS
)

type ServiceInfoLevel uint32

// Service Info Levels
const (
	ScEnumProcessInfo ServiceInfoLevel = C.SC_ENUM_PROCESS_INFO
)

// Contains process status information for a service.
type ServiceStatusProcess C.SERVICE_STATUS_PROCESS

// Contains the name of a service in a service control manager database and information about the service.
type EnumServiceStatusProcess C.ENUM_SERVICE_STATUS_PROCESS
