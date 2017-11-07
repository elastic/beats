// +build linux,cgo

package linux

/*
#include <unistd.h>
*/
import "C"

// GetClockTicks returns the number of click ticks in one jiffie.
func GetClockTicks() int {
	return int(C.sysconf(C._SC_CLK_TCK))
}
