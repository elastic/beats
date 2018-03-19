// +build ignore

package service

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
	SERVICE_ERROR_INVALID_NAME            ServiceErrno = C.ERROR_INVALID_NAME
	SERVICE_ERROR_SHUTDOWN_IN_PROGRESS    ServiceErrno = C.ERROR_SHUTDOWN_IN_PROGRESS
	SERVICE_ERROR_DATABASE_DOES_NOT_EXIST ServiceErrno = C.ERROR_DATABASE_DOES_NOT_EXIST
	SERVICE_ERROR_INSUFFICIENT_BUFFER     ServiceErrno = C.ERROR_INSUFFICIENT_BUFFER
	SERVICE_ERROR_SERVICE_DOES_NOT_EXIST  ServiceErrno = C.ERROR_SERVICE_DOES_NOT_EXIST
)

type ServiceErrorControl uint32

// Service Error Controls
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
	ServiceRunning         ServiceState = C.SERVICE_RUNNING
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

type ServiceSCMAccessRight uint32

// Access Rights for the Service Control Manager
const (
	// Includes STANDARD_RIGHTS_REQUIRED, in addition to all access rights in this table.
	ScManagerAllAccess ServiceSCMAccessRight = C.SC_MANAGER_ALL_ACCESS
	// Required to connect to the service control manager.
	ScManagerConnect ServiceSCMAccessRight = C.SC_MANAGER_CONNECT
	// Required to call the EnumServicesStatus or EnumServicesStatusEx function to list the services that are in the database.
	ScManagerEnumerateService ServiceSCMAccessRight = C.SC_MANAGER_ENUMERATE_SERVICE
	// Required to call the QueryServiceLockStatus function to retrieve the lock status information for the database.
	ScManagerQueryLockStatus ServiceSCMAccessRight = C.SC_MANAGER_QUERY_LOCK_STATUS
)

type ServiceAccessRight uint32

// Access Rights for a Service
const (
	// Includes STANDARD_RIGHTS_REQUIRED in addition to all access rights in this table.
	ServiceAllAccess ServiceAccessRight = C.SERVICE_ALL_ACCESS
	// Required to call the ChangeServiceConfig or ChangeServiceConfig2 function to change the service configuration. Because this grants the caller the right to change the executable file that the system runs, it should be granted only to administrators.
	ServcieChangeConfig ServiceAccessRight = C.SERVICE_CHANGE_CONFIG
	// Required to call the EnumDependentServices function to enumerate all the services dependent on the service.
	ServiceEnumerateDependents ServiceAccessRight = C.SERVICE_ENUMERATE_DEPENDENTS
	// Required to call the ControlService function to ask the service to report its status immediately.
	ServiceInterrogate ServiceAccessRight = C.SERVICE_INTERROGATE
	// Required to call the ControlService function to pause or continue the service.
	ServicePauseContinue ServiceAccessRight = C.SERVICE_PAUSE_CONTINUE
	// Required to call the QueryServiceConfig and QueryServiceConfig2 functions to query the service configuration.
	ServiceQueryConfig ServiceAccessRight = C.SERVICE_QUERY_CONFIG
	// Required to call the QueryServiceStatus or QueryServiceStatusEx function to ask the service control manager about the status of the service.
	ServiceQueryStatus ServiceAccessRight = C.SERVICE_QUERY_STATUS
	// Required to call the StartService function to start the service.
	ServiceStart ServiceAccessRight = C.SERVICE_START
	// Required to call the ControlService function to stop the service.
	ServiceStop ServiceAccessRight = C.SERVICE_STOP
	// Required to call the ControlService function to specify a user-defined control code.
	ServiceUserDefinedControl ServiceAccessRight = C.SERVICE_USER_DEFINED_CONTROL
)

type ServiceInfoLevel uint32

// Service Info Levels
const (
	ScEnumProcessInfo ServiceInfoLevel = C.SC_ENUM_PROCESS_INFO
)

type ServiceStartType uint32

// Service Start Types
const (
	// A service started automatically by the service control manager during system startup.
	ServiceAutoStart ServiceStartType = C.SERVICE_AUTO_START
	// A device driver started by the system loader. This value is valid only for driver services.
	ServiceBootStart ServiceStartType = C.SERVICE_BOOT_START
	// A service started by the service control manager when a process calls the StartService function.
	ServiceDemandStart ServiceStartType = C.SERVICE_DEMAND_START
	// A service that cannot be started. Attempts to start the service result in the error code ERROR_SERVICE_DISABLED.
	ServiceDisabled ServiceStartType = C.SERVICE_DISABLED
	// A device driver started by the IoInitSystem function. This value is valid only for driver services.
	ServiceSystemStart ServiceStartType = C.SERVICE_SYSTEM_START
)

type ProcessAccessRight uint32

const (
	ProcessAllAccess             ProcessAccessRight = C.PROCESS_ALL_ACCESS
	ProcessCreateProcess         ProcessAccessRight = C.PROCESS_CREATE_PROCESS
	ProcessCreateThread          ProcessAccessRight = C.PROCESS_CREATE_THREAD
	ProcessDupHandle             ProcessAccessRight = C.PROCESS_DUP_HANDLE
	ProcessQueryInformation      ProcessAccessRight = C.PROCESS_QUERY_INFORMATION
	ProcessQueryLimitInformation ProcessAccessRight = C.PROCESS_QUERY_LIMITED_INFORMATION
	ProcessSetInformation        ProcessAccessRight = C.PROCESS_SET_INFORMATION
	ProcessSetQuota              ProcessAccessRight = C.PROCESS_SET_QUOTA
	ProcessSuspendResume         ProcessAccessRight = C.PROCESS_SUSPEND_RESUME
	ProcessTerminate             ProcessAccessRight = C.PROCESS_TERMINATE
	ProcessVmOperation           ProcessAccessRight = C.PROCESS_VM_OPERATION
	ProcessVmRead                ProcessAccessRight = C.PROCESS_VM_READ
	ProcessVmWrite               ProcessAccessRight = C.PROCESS_VM_WRITE
	ProcessSynchronize           ProcessAccessRight = C.SYNCHRONIZE
)

// Contains process status information for a service.
type ServiceStatusProcess C.SERVICE_STATUS_PROCESS

// Contains the name of a service in a service control manager database and information about the service.
type EnumServiceStatusProcess C.ENUM_SERVICE_STATUS_PROCESS

//Contains configuration information for an installed service.
type QueryServiceConfig C.QUERY_SERVICE_CONFIG
