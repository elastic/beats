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

type ServiceErrorControl uint32

// Servcie Error Controls
const (
	SERVICE_ERROR_CRITICAL ServiceErrno = C.SERVICE_ERROR_CRITICAL
	SERVICE_ERROR_IGNORE   ServiceErrno = C.SERVICE_ERROR_IGNORE
	SERVICE_ERROR_NORMAL   ServiceErrno = C.SERVICE_ERROR_NORMAL
	SERVICE_ERROR_SEVERE   ServiceErrno = C.SERVICE_ERROR_SEVERE
)

var serviceErrors = map[ServiceErrno]struct{}{
	SERVICE_ERROR_ACCESS_DENIED:           struct{}{},
	SERVICE_ERROR_MORE_DATA:               struct{}{},
	SERVICE_ERROR_INVALID_PARAMETER:       struct{}{},
	SERVICE_ERROR_INVALID_HANDLE:          struct{}{},
	SERVICE_ERROR_INVALID_LEVEL:           struct{}{},
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS:    struct{}{},
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST: struct{}{},
	SERVICE_ERROR_CRITICAL:                struct{}{},
	SERVICE_ERROR_IGNORE:                  struct{}{},
	SERVICE_ERROR_NORMAL:                  struct{}{},
	SERVICE_ERROR_SEVERE:                  struct{}{},
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
	ServiceWin32Shareprocess  ServiceType = C.SERVICE_WIN32_SHARE_PROCESS
	ServiceInteractiveProcess ServiceType = C.SERVICE_INTERACTIVE_PROCESS
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

type ServcieStartType uint32

// Service Start Types
const (
	// A service started automatically by the service control manager during system startup.
	ServiceAuotStart ServcieStartType = C.SERVICE_AUTO_START
	// A device driver started by the system loader. This value is valid only for driver services.
	ServiceBootStart ServcieStartType = C.SERVICE_BOOT_START
	// A service started by the service control manager when a process calls the StartService function.
	ServiceDemandStart ServcieStartType = C.SERVICE_DEMAND_START
	// A service that cannot be started. Attempts to start the service result in the error code ERROR_SERVICE_DISABLED.
	ServiceDisabled ServcieStartType = C.SERVICE_DISABLED
	// A device driver started by the IoInitSystem function. This value is valid only for driver services.
	ServcieSystemStart ServcieStartType = C.SERVICE_SYSTEM_START
)

// Contains process status information for a service.
type ServiceStatusProcess C.SERVICE_STATUS_PROCESS

// Contains the name of a service in a service control manager database and information about the service.
type EnumServiceStatusProcess C.ENUM_SERVICE_STATUS_PROCESS
