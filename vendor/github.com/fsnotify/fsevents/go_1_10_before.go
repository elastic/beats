// +build darwin,!go1.10

package fsevents

/*
#include <CoreServices/CoreServices.h>
*/
import "C"

var (
	nullCFStringRef = C.CFStringRef(nil)
	nullCFUUIDRef   = C.CFUUIDRef(nil)
)

// NOTE: The following code is identical between go_1_10_after and go_1_10_before.

// GetDeviceUUID retrieves the UUID required to identify an EventID
// in the FSEvents database
func GetDeviceUUID(deviceID int32) string {
	uuid := C.FSEventsCopyUUIDForDevice(C.dev_t(deviceID))
	if uuid == nullCFUUIDRef {
		return ""
	}
	return cfStringToGoString(C.CFUUIDCreateString(C.kCFAllocatorDefault, uuid))
}
