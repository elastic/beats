// +build darwin,go1.10

package fsevents

/*
#include <CoreServices/CoreServices.h>
*/
import "C"

const (
	nullCFStringRef = C.CFStringRef(0)
	nullCFUUIDRef   = C.CFUUIDRef(0)
)

// NOTE: The following code is identical between go_1_10_after and go_1_10_before,
// however versions of Go 1.10.x prior to 1.10.4 fail to compile when the code utilizing
// the above constants is in a different file (wrap.go).

// GetDeviceUUID retrieves the UUID required to identify an EventID
// in the FSEvents database
func GetDeviceUUID(deviceID int32) string {
	uuid := C.FSEventsCopyUUIDForDevice(C.dev_t(deviceID))
	if uuid == nullCFUUIDRef {
		return ""
	}
	return cfStringToGoString(C.CFUUIDCreateString(C.kCFAllocatorDefault, uuid))
}
